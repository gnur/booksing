package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/cheggaaa/pb/v3"
	"github.com/gnur/booksing"
	"github.com/jaffee/commandeer"
	"github.com/mattn/go-zglob"
	"github.com/minio/minio-go"
	log "github.com/sirupsen/logrus"
)

var client http.Client

var accessKeyID = os.Getenv("ACCESS_KEY_ID")
var secretAccessKey = os.Getenv("SECRET_ACCESS_KEY")

var mc *minio.Client

// Configuration bla
type Configuration struct {
	Bucket       string `help:"What bucket is used to store the lambda code zips?"`
	Host         string `help:"Hostname of S3 compatible API"`
	ImportDir    string `help:"Directory to load books from"`
	BooksingHost string `help:"FQDN of the main booksing server"`
	APIKey       string `help:"api key for booksing"`
	Debug        bool   `help:"Enable debug mode?"`
}

func newConfig() *Configuration {
	return &Configuration{
		ImportDir:    ".",
		BooksingHost: "localhost:8080",
	}
}

var runtimes = []string{
	"go1.x",
}

var cfg Configuration

func main() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "15:04:05.999"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	err := commandeer.Run(newConfig())
	if err != nil {
		log.WithField("err", err).Error("failed")
	}
	log.SetLevel(log.DebugLevel)

}

// Run does the actual thingies
func (cfg *Configuration) Run() error {
	errors := false
	if cfg.Bucket == "" {
		log.Error("please provide the bucket name to upload to")
		errors = true
	}
	if cfg.Host == "" {
		log.Error("please provide the S3 hostname")
		errors = true
	}
	if errors {
		log.Fatal("Invalid information provided")
	}

	log.Info("Starting import")
	cfg.Import()

	return nil
}

// Import indexes the directory, uploads to S3 and announces it to booksing
func (cfg *Configuration) Import() {
	matches, err := zglob.Glob(filepath.Join(cfg.ImportDir, "/**/*.epub"))
	if err != nil {
		log.WithField("err", err).Error("glob of all books failed")
		return
	}
	if len(matches) == 0 {
		log.Info("finished refresh of booklist, no new books found")
		return
	}

	var bar *pb.ProgressBar

	if !cfg.Debug {
		bar = pb.StartNew(len(matches))
	}

	var result struct {
		Invalid    uint
		Uploaded   uint
		Duplicates uint
		Errors     uint
	}

	var booksToAdd []booksing.BookInput

	for _, bookPath := range matches {
		if !cfg.Debug {
			bar.Increment()
		}
		book, err := booksing.NewBookFromFile(bookPath, false, "")
		if err != nil {
			result.Invalid++
			continue
		}

		found, err := cfg.bookInBooksing(book.Author, book.Title)
		if err != nil {
			result.Errors++
			continue
		}
		if found {
			result.Duplicates++
			continue
		}
		// book is valid, and not in booksing

		key, err := cfg.UploadToS3(bookPath, book)
		if err != nil {
			result.Errors++
			continue
		}

		booksToAdd = append(booksToAdd, cfg.getBooksingInput(book, key))

		result.Uploaded++
	}
	if !cfg.Debug {
		bar.Finish()
	}

	if len(booksToAdd) > 0 {
		log.Info("Starting announcement of books to booksing")
		batchSize := 100
		pos := 0
		total := len(booksToAdd)
		for {
			start := pos
			end := pos + batchSize
			if end > total {
				end = total
			}
			batch := booksToAdd[start:end]
			cfg.addBooksToBooksing(batch)
			if end == total {
				break
			}
			pos += batchSize
		}

	}

	pretty, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(pretty))
	log.Info("done")

}

func (cfg *Configuration) bookInBooksing(author, title string) (bool, error) {
	url := fmt.Sprintf("%s/api/exists/%s/%s",
		cfg.BooksingHost,
		url.QueryEscape(author),
		url.QueryEscape(title))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		// return true to be safe
		return true, err
	}
	req.Header.Add("x-api-key", cfg.APIKey)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 403 {
			log.Fatal("access denied")
		}
		// return true to be safe
		return true, err
	}
	defer resp.Body.Close()

	result, _ := ioutil.ReadAll(resp.Body)

	var found map[string]bool

	err = json.Unmarshal(result, &found)
	if err != nil {
		// return true to be safe
		return true, err
	}

	if _, ok := found["found"]; ok {
		return found["found"], nil
	}

	// default return true
	return true, errors.New("found not found in response")
}

func (cfg *Configuration) UploadToS3(bookpath string, book *booksing.Book) (string, error) {
	var err error
	if mc == nil {
		mc, err = minio.New(cfg.Host, accessKeyID, secretAccessKey, true)
		if err != nil {
			log.WithField("err", err).Fatal("creating minio client failed, exiting hard")
		}
	}

	key := booksing.GetBookPath(book.Title, book.Author) + ".epub"
	_, err = mc.FPutObject(cfg.Bucket, key, bookpath, minio.PutObjectOptions{})
	if err != nil {
		log.WithField("err", err).Error("could not upload")
	}

	return key, err
}

func (cfg *Configuration) getBooksingInput(b *booksing.Book, key string) booksing.BookInput {
	return booksing.BookInput{
		Title:       b.Title,
		Author:      b.Author,
		Language:    b.Language,
		Description: b.Description,
		Locations: map[string]booksing.Location{
			"epub": booksing.Location{
				Type: booksing.S3Storage,
				S3: &booksing.S3Location{
					Host:   cfg.Host,
					Bucket: cfg.Bucket,
					Key:    key,
				},
			},
		},
	}
}

func (cfg *Configuration) addBookToBooksing(b *booksing.Book, key string) error {
	url := fmt.Sprintf("%s/api/book",
		cfg.BooksingHost)

	in := booksing.BookInput{
		Title:       b.Title,
		Author:      b.Author,
		Language:    b.Language,
		Description: b.Description,
		Locations: map[string]booksing.Location{
			"epub": booksing.Location{
				Type: booksing.S3Storage,
				S3: &booksing.S3Location{
					Host:   cfg.Host,
					Bucket: cfg.Bucket,
					Key:    key,
				},
			},
		},
	}

	js, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(js))
	if err != nil {
		// return true to be safe
		return err
	}
	req.Header.Add("x-api-key", cfg.APIKey)
	req.Header.Add("Contenty-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 403 {
			log.Fatal("access denied")
		}
		return err
	}
	resp.Body.Close()

	return nil
}

func (cfg *Configuration) addBooksToBooksing(batch []booksing.BookInput) error {
	url := fmt.Sprintf("%s/api/books",
		cfg.BooksingHost)

	js, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(js))
	if err != nil {
		// return true to be safe
		return err
	}
	req.Header.Add("x-api-key", cfg.APIKey)
	req.Header.Add("Contenty-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 403 {
			log.Fatal("access denied")
		}
		return err
	}
	resp.Body.Close()

	return nil
}
