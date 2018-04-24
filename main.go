package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/globalsign/mgo/bson"

	"github.com/globalsign/mgo"
	"github.com/google/uuid"
	"github.com/jordan-wright/email"
	zglob "github.com/mattn/go-zglob"
)

type booksingApp struct {
	books *mgo.Collection
	users *mgo.Collection
}

type bookResponse struct {
	Books      []Book `json:"books"`
	TotalCount int    `json:"total"`
	timestamp  time.Time
}

type bookConvertRequest struct {
	BookID        string `json:"bookid"`
	Receiver      string `json:"email"`
	SMTPServer    string `json:"smtpserver"`
	SMTPUser      string `json:"smtpuser"`
	SMTPPassword  string `json:"smtppass"`
	ConvertToMobi bool   `json:"convert"`
}

// User represents a user..
type User struct {
	Name  string
	Token string
}

func main() {
	envDeletes := os.Getenv("ALLOW_DELETES")
	allowDeletes := envDeletes != "" && strings.ToLower(envDeletes) == "true"
	bookDir := os.Getenv("BOOK_DIR")
	if bookDir == "" {
		bookDir = "."
	}
	mongoHost := os.Getenv("MONGO_HOST")
	if mongoHost == "" {
		mongoHost = "localhost"
	}
	fmt.Println("Connecting to", mongoHost)
	conn, err := mgo.Dial(mongoHost)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("Connected")
	session := conn.DB("booksing")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	app := booksingApp{
		books: session.C("books"),
		users: session.C("users"),
	}

	http.HandleFunc("/convert", func(w http.ResponseWriter, r *http.Request) {
		var convert bookConvertRequest
		if r.Body == nil {
			http.Error(w, "please provide body", 400)
			return
		}
		err := json.NewDecoder(r.Body).Decode(&convert)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var convertBook Book
		//err = db.One("ID", convert.BookID, &convertBook)
		err = app.books.Find(bson.M{"id": convert.BookID}).One(&convertBook)
		if err == nil {
			go convertAndSendBook(&convertBook, convert)
		} else {
			log.Println(err.Error())
		}
		log.Println(convert.BookID)
	})
	http.HandleFunc("/refresh", app.refreshBooks(bookDir, allowDeletes))
	http.HandleFunc("/books.json", app.getBooks())
	http.HandleFunc("/adduser", app.addUser())
	http.HandleFunc("/download/", app.getBook())

	http.Handle("/", http.FileServer(assetFS()))
	log.Println("Please visit http://localhost:7132 to view booksing")
	log.Fatal(http.ListenAndServe(":7132", nil))
}

func convertAndSendBook(c *Book, req bookConvertRequest) {
	var attachment string
	log.Println("-----------------------------------")
	if !c.HasMobi && req.ConvertToMobi {
		log.Println("first convert the book")
		cmd := exec.Command("kindlegen", c.Filepath)
		log.Printf("Running command and waiting for it to finish...")
		err := cmd.Run()
		if err != nil {
			log.Printf("Command finished with error: %v", err)
			mobiPath := strings.Replace(c.Filepath, ".epub", ".mobi", 1)
			cmd := exec.Command("ebook-convert", c.Filepath, mobiPath)
			log.Printf("Running command and waiting for it to finish...")
			err := cmd.Run()
			if err != nil {
				log.Printf("Command finished with error: %v", err)
			} else {
				c.HasMobi = true
			}
		} else {
			c.HasMobi = true
		}
	}
	if c.HasMobi && req.ConvertToMobi {
		attachment = strings.Replace(c.Filepath, ".epub", ".mobi", 1)
	} else if !req.ConvertToMobi {
		attachment = c.Filepath
	} else {
		log.Println("mobi not present but was requested")
		return
	}
	e := email.NewEmail()
	e.From = req.SMTPUser
	e.To = []string{req.Receiver}
	e.Subject = "A booksing book"
	e.Text = []byte("")
	e.AttachFile(attachment)
	err := e.Send(req.SMTPServer+":587", smtp.PlainAuth("", req.SMTPUser, req.SMTPPassword, req.SMTPServer))
	if err != nil {
		log.Println(err.Error())
	}

	log.Println(c)
	log.Println("-----------------------------------")
}
func (app booksingApp) getBook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("token")
		if err != nil || !app.validToken(c.Value) {
			http.Error(w, "denied", http.StatusForbidden)
			return
		}
		fileName := r.URL.Query().Get("book")
		fmt.Println(c.Value)
		fmt.Println("trying to download ", fileName)
		var book Book
		err = app.books.Find(bson.M{"filename": fileName}).One(&book)
		if err != nil {
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", path.Base(book.Filepath)))
		http.ServeFile(w, r, book.Filepath)
	}
}

func (app booksingApp) validToken(cookie string) bool {
	if os.Getenv("TOKEN_REQUIRED") != "true" {
		return true
	}
	parts := strings.SplitN(cookie, "_", 2)
	if len(parts) != 2 {
		return false
	}
	var user User
	err := app.users.Find(bson.M{"name": parts[0]}).One(&user)
	if err != nil {
		return false
	}
	return parts[1] == user.Token
}

func (app booksingApp) addUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addr := strings.Split(r.RemoteAddr, ":")[0]
		if addr != "127.0.0.1" {
			http.Error(w, "denied", http.StatusForbidden)
			return
		}
		username := r.URL.Query().Get("username")
		if username == "" {
			http.Error(w, "please provide username", http.StatusNotFound)
			return
		}
		token := uuid.New().String()
		var user User
		user.Name = username
		user.Token = token
		err := app.users.Insert(&user)
		if err != nil {
			http.Error(w, "unknown error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		io.WriteString(w, fmt.Sprintf("%s_%s\n", username, token))
	}
}

func (app booksingApp) getBooks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("token")
		if err != nil || !app.validToken(c.Value) {
			http.Error(w, "denied", http.StatusForbidden)
			return
		}
		var resp bookResponse
		var limit int
		numString := r.URL.Query().Get("results")
		filter := strings.ToLower(r.URL.Query().Get("filter"))
		limit = 1000
		if a, err := strconv.Atoi(numString); err == nil {
			if a > 0 && a < 1000 {
				limit = a
			}
		}
		numResults, err := app.books.Count()
		if err != nil {
			log.Println(err)
		}
		resp.TotalCount = numResults

		resp.Books = app.filterBooks(filter, limit)
		if len(resp.Books) == 0 {
			resp.Books = []Book{}
		}

		json.NewEncoder(w).Encode(resp)
	}
}

func (app booksingApp) filterBooks(filter string, limit int) []Book {
	var books []Book
	var iter *mgo.Iter
	if filter == "" {
		iter = app.books.Find(nil).Limit(limit).Iter()
	} else {
		iter = app.books.Find(bson.M{"author": bson.RegEx{Pattern: filter}}).Limit(limit).Iter()
		//TODO: add title
	}
	err := iter.All(&books)
	if err != nil {
		log.Println(err.Error())
		return []Book{}
	}
	return books
}

func (app booksingApp) refreshBooks(bookDir string, allowDeletes bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("starting refresh of booklist")
		matches, err := zglob.Glob(filepath.Join(bookDir, "/**/*.epub"))
		if err != nil {
			fmt.Println("Scan could not complete: ", err.Error())
			return
		}
		log.Println("found", len(matches), "epubs in ", bookDir)

		bookQ := make(chan string, len(matches))
		resultQ := make(chan int)

		for w := 0; w < 1; w++ { //not sure yet how concurrent-proof my solution is
			go app.bookParser(bookQ, resultQ, allowDeletes)
		}

		for _, filename := range matches {
			bookQ <- filename
		}

		for a := 0; a < len(matches); a++ {
			<-resultQ
			if a > 0 && a%100 == 0 {
				fmt.Println("Scraped", a, "books so far")
			}
		}

		log.Println("started refresh of booklist")
	}
}

func (app booksingApp) bookParser(bookQ chan string, resultQ chan int, allowDeletes bool) {
	for filename := range bookQ {
		log.Println("parsing", filename)
		var dbBook Book
		//err := db.One("Filepath", filename, &dbBook)
		err := app.books.Find(bson.M{"filepath": filename}).One(&dbBook)
		if err == nil {
			resultQ <- 1
			continue
		}
		book, err := NewBookFromFile(filename)
		if err != nil {
			if allowDeletes {
				fmt.Println("Deleting ", filename)
				os.Remove(filename)
			}
			resultQ <- 1
			continue
		}
		//err = db.One("ID", book.ID, &dbBook)
		err = app.books.Find(bson.M{"hash": book.Hash}).One(&dbBook)
		if err != nil {
			//TODO: find out what happens if One() fails
			book.ID = bson.NewObjectId()
			err = app.books.Insert(book)
			if err != nil {
				fmt.Println(err)
			}
		} else if allowDeletes {
			fmt.Println("Deleting ", filename)
			os.Remove(filename)
		}
		//for _, tag := range book.MatchKey {
		//	addBookToTag(db, tag, book.ID)
		//}
		resultQ <- 1
	}
}
