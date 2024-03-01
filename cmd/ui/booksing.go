package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gnur/booksing"
	zglob "github.com/mattn/go-zglob"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

const (
	stateUnlocked uint32 = iota
	stateLocked
)

var (
	locker = stateUnlocked
)

func (app *booksingApp) refreshLoop() {
	for {
		app.refresh()
		time.Sleep(time.Minute)
	}
}

func (app *booksingApp) cover(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=86400, immutable")

	//join the path with a slash to make sure it is an absolute path
	//and the Join will also automatically clean out any path traversal characters
	file := path.Join("/", c.Query("file"))

	//join only with the bookDir after the first join so only files from the bookdir are served
	file = path.Join(app.bookDir, file)

	c.File(file)
}

func (app *booksingApp) downloadBook(c *gin.Context) {

	hash := c.Query("hash")
	file := c.Query("file")

	book, err := app.searchDB.GetBook(hash)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"err":  err,
			"hash": hash,
		}).Error("could not find book")
		return
	}
	user := c.MustGet("id")
	username := user.(*booksing.User).Name

	ip := c.ClientIP()
	dl := booksing.Download{
		User:      username,
		IP:        ip,
		Book:      book.Hash,
		Timestamp: time.Now(),
	}
	err = app.db.AddDownload(dl)
	if err != nil {
		app.logger.WithField("err", err).Error("could not store download")
	}

	if file != "" {
		fName := path.Base(file)
		c.Header("Content-Disposition",
			fmt.Sprintf("attachment; filename=\"%s\"", fName))
		c.File(file)
	} else {
		fName := path.Base(book.Path)
		c.Header("Content-Disposition",
			fmt.Sprintf("attachment; filename=\"%s\"", fName))
		c.File(book.Path)
	}
}

func (app *booksingApp) updateUser(c *gin.Context) {
	id := c.Param("username")
	dbUser, err := app.db.GetUser(id)
	if err != nil {
		app.logger.WithField("err", err).Error("Could not get user")
		c.HTML(500, "error.html", V{
			Error: err,
		})
		c.Abort()
		return
	}
	var u booksing.User
	if err := c.ShouldBind(&u); err != nil {
		app.logger.WithField("err", err).Warning("could not get values from post")
		c.HTML(400, "error.html", V{
			Error: err,
		})
		c.Abort()
		return
	}

	dbUser.IsAllowed = u.IsAllowed
	err = app.db.SaveUser(&dbUser)
	if err != nil {
		app.logger.WithField("err", err).Error("Could not update user")
		c.JSON(500, gin.H{
			"text": "could not find user",
		})
		c.Abort()
	}

	c.Redirect(302, c.Request.Referer())
}

func (app *booksingApp) refresh() {
	if !atomic.CompareAndSwapUint32(&locker, stateUnlocked, stateLocked) {
		app.logger.Warning("not refreshing because it is already running")
		return
	}
	defer atomic.StoreUint32(&locker, stateUnlocked)
	defer func() {
		app.state = "idle"
	}()

	app.state = "indexing"
	matches, err := zglob.Glob(filepath.Join(app.importDir, "/**/*.epub"))
	if err != nil {
		app.logger.WithField("err", err).Error("glob of all books failed")
		return
	}

	if len(matches) == 0 {
		app.logger.Debug("Not adding any books because nothing is new")
		return
	}
	var books []booksing.Book
	counter := 0

	app.logger.WithFields(logrus.Fields{
		"total":   len(matches),
		"bookdir": app.importDir,
	}).Info("located books on filesystem, processing per batchsize")
	ctx := context.TODO()
	toProcess := len(matches)
	bookQ := make(chan *booksing.Book)
	sem := semaphore.NewWeighted(int64(runtime.GOMAXPROCS(0)))

	for _, filename := range matches {
		app.logger.WithField("f", filename).Debug("parsing book")

		go func(f string) {
			if err := sem.Acquire(ctx, 1); err != nil {
				app.logger.WithError(err).Error("failed to acquire semaphore")
			}
			defer sem.Release(1)

			book, err := booksing.NewBookFromFile(f, app.bookDir)
			if err != nil {
				app.logger.WithError(err).Error("Failed to parse book")
				app.moveBookToFailed(f)
			}

			bookQ <- book
		}(filename)

	}
	processed := 0
	for book := range bookQ {
		processed++
		if book == nil {
			if processed == toProcess {
				close(bookQ)
			}
			continue
		}
		if !app.keepBook(book) {
			app.moveBookToFailed(book.Path)
			if processed == toProcess {
				close(bookQ)
			}
			continue
		}
		books = append(books, *book)
		counter++
		if len(books) == 50 || processed == toProcess {
			err = app.searchDB.AddBooks(books)
			if err != nil {
				app.logger.WithFields(logrus.Fields{
					"err": err,
				}).Warning("bulk insert into meili failed")
			}
			books = []booksing.Book{}
		}
		app.logger.WithField("counter", counter).Debug("Found some books")
		if processed == toProcess {
			close(bookQ)
		}

	}
	if len(books) > 0 {
		app.logger.Error("This should absolutely not happen")
		err = app.searchDB.AddBooks(books)
		if err != nil {
			app.logger.WithFields(logrus.Fields{
				"reason": "bulk insert failed",
				"err":    err,
			}).Info("bulk insert failed")
		}
	}

	app.logger.Info("Done with refresh")

	//move none epub files to failed dir

	//remove empty directories

}

func (app *booksingApp) moveBookToFailed(bookpath string) {
	err := os.MkdirAll(app.cfg.FailDir, 0755)
	if err != nil {
		app.logger.WithError(err).Error("unable to create fail dir")
		return
	}
	globPath := strings.Replace(bookpath, ".epub", ".*", 1)
	app.logger.WithField("path", globPath).Debug("Searching here for other formats")
	files, err := zglob.Glob(globPath)
	if err != nil {
		app.logger.WithError(err).Error("failed to glob relavent files")
		return
	}

	for _, f := range files {
		filename := path.Base(f)
		newBookPath := path.Join(app.cfg.FailDir, filename)
		err = os.Rename(bookpath, newBookPath)
		if err != nil {
			app.logger.WithFields(logrus.Fields{
				"faildir": app.cfg.FailDir,
				"book":    bookpath,
			}).WithError(err).Error("unable to move book to faildir")
		}
	}
}

func (app *booksingApp) addUser(c *gin.Context) {
	var u booksing.User
	if err := c.ShouldBind(&u); err != nil {
		app.logger.WithField("err", err).Warning("could not get values from post")
		c.HTML(400, "error.html", V{
			Error: err,
		})
		c.Abort()
		return
	}

	u.Created = time.Now().In(app.timezone)
	err := app.db.SaveUser(&u)
	if err != nil {
		app.logger.WithField("err", err).Error("Could not update user")
		c.JSON(500, gin.H{
			"text": "could not find user",
		})
		c.Abort()
	}

	c.Redirect(302, c.Request.Referer())
}
