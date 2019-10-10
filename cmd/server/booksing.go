package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gnur/booksing"
	zglob "github.com/mattn/go-zglob"
	"github.com/sirupsen/logrus"
)

const (
	stateUnlocked uint32 = iota
	stateLocked
)

var (
	locker    = stateUnlocked
	errLocked = errors.New("already running")
)

func (app *booksingApp) refreshLoop() {
	for {
		app.refresh()
		time.Sleep(time.Hour)
	}
}

func (app *booksingApp) downloadBook(c *gin.Context) {

	hash := c.Query("hash")
	index := c.Query("index")

	book, err := app.db.GetBookBy("Hash", hash)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"err":  err,
			"hash": hash,
		}).Error("could not find book")
		return
	}

	ip := c.ClientIP()
	dl := booksing.Download{
		//			User:      r.Header.Get("x-auth-user"),
		User:      "unknown",
		IP:        ip,
		Book:      book.Hash,
		Timestamp: time.Now(),
	}
	err = app.db.AddDownload(dl)
	if err != nil {
		app.logger.WithField("err", err).Error("could not store download")
	}

	fileLocation, exists := book.Locations[index]
	if !exists {
		app.logger.WithFields(logrus.Fields{
			"hash":  hash,
			"index": index,
		}).Warning("invalid index provided")
		c.JSON(404, gin.H{
			"text": "file not found",
		})
		return
	}

	if fileLocation.Type == booksing.FileStorage && fileLocation.File != nil {
		c.Header("Content-Disposition",
			fmt.Sprintf("attachment; filename=\"%s\"", path.Base(fileLocation.File.Path)))
		c.File(fileLocation.File.Path)
		return
	}
}

func (app *booksingApp) bookPresent(c *gin.Context) {
	author := c.Param("author")
	title := c.Param("title")
	title = booksing.Fix(title, true, false)
	author = booksing.Fix(author, true, true)
	hash := booksing.HashBook(author, title)

	_, err := app.db.GetBookBy("Hash", hash)
	found := err == nil

	c.JSON(200, map[string]bool{"found": found})
}

func (app *booksingApp) getBook(c *gin.Context) {
	hash := c.Param("hash")
	book, err := app.db.GetBookBy("Hash", hash)
	if err != nil {
		return
	}
	c.JSON(200, book)
}

func (app *booksingApp) getUser(c *gin.Context) {
	//admin := app.userIsAdmin(r)
	admin := true
	c.JSON(200, gin.H{
		"admin": admin,
	})
}

func (app *booksingApp) userIsAdmin(r *http.Request) bool {
	user := r.Header.Get("x-auth-user")
	admin := false
	if user == os.Getenv("ADMIN_USER") || os.Getenv("ANONYMOUS_ADMIN") != "" {
		admin = true
	}
	app.logger.WithFields(logrus.Fields{
		"x-auth-user": user,
		"admin":       admin,
		"env-user":    os.Getenv("ADMIN_USER"),
		"anon-admin":  os.Getenv("ANONYMOUS_ADMIN"),
	}).Info("getting user admin")
	return admin

}

func (app *booksingApp) getDownloads(c *gin.Context) {
	//admin := app.userIsAdmin(r)
	admin := true
	if !admin {
		c.JSON(403, gin.H{
			"text": "access denied",
		})
		return
	}
	downloads, _ := app.db.GetDownloads(200)
	c.JSON(200, downloads)
}

func (app *booksingApp) getRefreshes(c *gin.Context) {
	//admin := app.userIsAdmin(r)
	admin := true
	if !admin {
		c.JSON(403, gin.H{
			"text": "access denied",
		})
		return
	}
	refreshes, _ := app.db.GetRefreshes(200)
	c.JSON(200, refreshes)
}

func (app *booksingApp) convertBook(c *gin.Context) {
	var req deleteRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, gin.H{
			"text": err.Error(),
		})
		return
	}
	app.logger.WithField("req", req).Info("got delete request")
	hash := req.Hash

	book, err := app.db.GetBookBy("Hash", hash)
	if err != nil {
		c.JSON(500, gin.H{
			"text": err.Error(),
		})
		return
	}
	//TODO: fix this
	app.logger.WithField("book", book.Hash).Debug("converting to mobi")
	mobiPath := strings.Replace(book.Hash, ".epub", ".mobi", 1)
	cmd := exec.Command("ebook-convert", book.Hash, mobiPath)

	_, err = cmd.CombinedOutput()
	if err != nil {
		app.logger.WithField("err", err).Error("Command finished with error")
		c.JSON(500, gin.H{
			"text": err.Error(),
		})
		return
	}

	app.db.SetBookConverted(hash)
	app.logger.WithField("book", book.Hash).Debug("conversion successful")
	c.JSON(200, book)
}

func (app *booksingApp) getBooks(c *gin.Context) {
	var resp bookResponse
	var limit int
	numString := c.DefaultQuery("results", "100")
	filter := strings.ToLower(c.Query("filter"))
	filter = strings.TrimSpace(filter)
	limit = 1000

	app.logger.WithFields(logrus.Fields{
		//"user":   r.Header.Get("x-auth-user"),
		"filter": filter,
	}).Info("user initiated search")

	if a, err := strconv.Atoi(numString); err == nil {
		if a > 0 && a < 1000 {
			limit = a
		}
	}
	resp.TotalCount = app.db.BookCount()

	books, err := app.db.GetBooks(filter, limit)
	if err != nil {
		app.logger.WithField("err", err).Error("error retrieving books")
	}
	resp.Books = books

	c.JSON(200, resp)
}

func (app *booksingApp) refresh() {
	if !atomic.CompareAndSwapUint32(&locker, stateUnlocked, stateLocked) {
		app.logger.Warning("not refreshing because it is already running")
		return
	}
	defer atomic.StoreUint32(&locker, stateUnlocked)
	app.logger.Info("starting refresh of booklist")
	results := booksing.RefreshResult{
		StartTime: time.Now(),
	}
	matches, err := zglob.Glob(filepath.Join(app.importDir, "/**/*.epub"))
	if err != nil {
		app.logger.WithField("err", err).Error("glob of all books failed")
		return
	}
	if len(matches) == 0 {
		app.logger.Info("finished refresh of booklist, no new books found")
		return
	}
	app.logger.WithFields(logrus.Fields{
		"total":   len(matches),
		"bookdir": app.bookDir,
	}).Info("located books on filesystem")

	bookQ := make(chan string, len(matches))
	resultQ := make(chan parseResult)

	for w := 0; w < 6; w++ { //not sure yet how concurrent-proof my solution is
		go app.bookParser(bookQ, resultQ)
	}

	for _, filename := range matches {
		bookQ <- filename
	}

	for a := 0; a < len(matches); a++ {
		r := <-resultQ

		switch r {
		case OldBook:
			results.Old++
		case InvalidBook:
			results.Invalid++
		case AddedBook:
			results.Added++
		case DuplicateBook:
			results.Duplicate++
		}
		if a > 0 && a%100 == 0 {
			app.logger.WithFields(logrus.Fields{
				"processed": a,
				"total":     len(matches),
			}).Info("processing books")
		}

	}
	total := app.db.BookCount()
	if err != nil {
		app.logger.WithField("err", err).Error("could not get total book count")
	}
	results.Old = total
	results.StopTime = time.Now()
	err = app.db.AddRefresh(results)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"err":     err,
			"results": results,
		}).Error("Could not save refresh results")
	}

	app.logger.WithField("result", results).Info("finished refresh of booklist")
}
func (app *booksingApp) refreshBooks(c *gin.Context) {
	app.refresh()
}

func (app *booksingApp) bookParser(bookQ chan string, resultQ chan parseResult) {
	for filename := range bookQ {
		_, err := app.db.GetBookBy("Filepath", filename)
		if err == nil {
			resultQ <- OldBook
			continue
		}
		book, err := booksing.NewBookFromFile(filename, app.allowOrganize, app.bookDir)
		if err != nil {
			if app.allowDeletes {
				app.logger.WithFields(logrus.Fields{
					"file":   filename,
					"reason": "invalid",
				}).Info("Deleting book")
				os.Remove(filename)
			}
			resultQ <- InvalidBook
			continue
		}
		err = app.db.AddBook(book)
		if err != nil {
			app.logger.WithFields(logrus.Fields{
				"file": filename,
				"err":  err,
			}).Error("could not store book")

			if err == booksing.ErrDuplicate {
				if app.allowDeletes {
					app.logger.WithFields(logrus.Fields{
						"file":   filename,
						"reason": "duplicate",
					}).Info("Deleting book")
					os.Remove(filename)
				}
				resultQ <- DuplicateBook
			}
		} else {
			resultQ <- AddedBook
		}
	}
}

func (app *booksingApp) addBook(c *gin.Context) {
	var b booksing.Book
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(400, gin.H{
			"text": "invalid input",
		})
		return
	}
}

type deleteRequest struct {
	Hash string `form:"hash"`
}

func (app *booksingApp) deleteBook(c *gin.Context) {
	//admin := app.userIsAdmin(r)
	admin := true
	if !admin {
		return
	}
	var req deleteRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, gin.H{
			"text": err.Error(),
		})
		return
	}
	app.logger.WithField("req", req).Info("got delete request")
	hash := req.Hash

	book, err := app.db.GetBookBy("Hash", hash)
	if err != nil {
		return
	}
	//TODO: fix this to use the new indexing
	if book.HasMobi() {
		mobiPath := strings.Replace(book.Hash, ".epub", ".mobi", 1)
		os.Remove(mobiPath)
	}
	os.Remove(book.Hash)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"hash": hash,
			"err":  err,
		}).Error("Could not delete book from filesystem")
		return
	}

	err = app.db.DeleteBook(hash)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"hash": hash,
			"err":  err,
		}).Error("Could not delete book from database")
		return
	}
	app.logger.WithFields(logrus.Fields{
		"hash": hash,
	}).Info("book was deleted")
	c.JSON(200, gin.H{
		"text": "ok",
	})
}
