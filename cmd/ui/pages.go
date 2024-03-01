package main

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gnur/booksing"
	zglob "github.com/mattn/go-zglob"
	"github.com/sirupsen/logrus"
)

func (app *booksingApp) searchAPI(c *gin.Context) {
	var offset int64
	var limit int64
	var err error
	offset = 0
	limit = 20
	q := c.Query("q")
	off := c.Query("o")
	if off != "" {
		offset, err = strconv.ParseInt(off, 10, 64)
		if err != nil {
			offset = 0
		}
	}
	lim := c.Query("l")
	if lim != "" {
		limit, err = strconv.ParseInt(lim, 10, 64)
		if err != nil {
			limit = 20
		}
	}

	var books *booksing.SearchResult

	books, err = app.searchDB.GetBooks(q, limit, offset)
	if err != nil {
		c.HTML(500, "error.html", V{
			Error: err,
			Q:     q,
		})
		return
	}

	c.JSON(200, books)
}

func (app *booksingApp) search(c *gin.Context) {
	start := time.Now()
	var offset int64
	var limit int64
	var err error
	offset = 0
	limit = 20
	q := c.Query("q")
	off := c.Query("o")
	if off != "" {
		offset, err = strconv.ParseInt(off, 10, 64)
		if err != nil {
			offset = 0
		}
	}
	lim := c.Query("l")
	if lim != "" {
		limit, err = strconv.ParseInt(lim, 10, 64)
		if err != nil {
			limit = 20
		}
	}

	var books *booksing.SearchResult

	books, err = app.searchDB.GetBooks(q, limit, offset)
	if err != nil {
		c.HTML(500, "error.html", V{
			Error: err,
			Q:     q,
		})
		return
	}

	stop := time.Since(start)
	latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))

	template := "search.html"
	if c.Request.Header.Get("HX-Request") == "true" {
		template = "searchresults"
	}

	c.HTML(200, template, V{
		Limit:      limit,
		Offset:     offset,
		Results:    books.Total,
		TimeTaken:  latency,
		Books:      books.Items,
		Error:      err,
		Q:          q,
		IsAdmin:    c.GetBool("isAdmin"),
		TotalBooks: app.searchDB.GetBookCount(),
		Indexing:   app.state == "indexing",
	})
}

func (app *booksingApp) showUsers(c *gin.Context) {

	users, err := app.db.GetUsers()
	if err != nil {
		c.HTML(403, "error.html", V{
			Error: err,
		})
		c.Abort()
		return
	}

	c.HTML(200, "users.html", V{
		Error:      err,
		Q:          "",
		IsAdmin:    c.GetBool("isAdmin"),
		TotalBooks: app.searchDB.GetBookCount(),
		Users:      users,
		Indexing:   app.state == "indexing",
	})

}

func (app *booksingApp) deleteBook(c *gin.Context) {
	hash := c.Param("hash")

	book, err := app.searchDB.GetBook(hash)
	if err != nil {
		c.HTML(404, "error.html", V{
			Error: errors.New("Book not found"),
		})
		return
	}

	err = os.Remove(book.Path)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"hash": hash,
			"err":  err,
			"path": book.Path,
		}).Error("Could not delete book from filesystem")
		c.HTML(500, "error.html", V{
			Error: fmt.Errorf("Unable to delete book from filesystem: %w", err),
		})
		return
	}

	err = app.searchDB.DeleteBook(hash)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"hash": hash,
			"err":  err,
		}).Error("Could not delete book from database")
		c.HTML(500, "error.html", V{
			Error: fmt.Errorf("Unable to delete book from database: %w", err),
		})
		return
	}
	app.logger.WithFields(logrus.Fields{
		"hash": hash,
	}).Info("book was deleted")
	c.Redirect(302, c.Request.Referer())
}

func (app *booksingApp) showDownloads(c *gin.Context) {
	dls, err := app.db.GetDownloads(100)
	if err != nil {
		c.HTML(403, "error.html", V{
			Error: err,
		})
		c.Abort()
		return
	}

	c.HTML(200, "downloads.html", V{
		Error:      err,
		Q:          "",
		IsAdmin:    c.GetBool("isAdmin"),
		TotalBooks: app.searchDB.GetBookCount(),
		Downloads:  dls,
		Indexing:   app.state == "indexing",
	})

}

func (app *booksingApp) detailPage(c *gin.Context) {
	hash := c.Param("hash")

	b, err := app.searchDB.GetBook(hash)
	if err != nil {
		slog.Error("could not find book", "err", err, "hash", hash)
		c.HTML(500, "error.html", V{
			Error: err,
		})
		return
	}

	globPath := strings.Replace(b.Path, ".epub", ".*", 1)
	app.logger.WithField("path", globPath).Debug("Searching here for other formats")
	books, err := zglob.Glob(globPath)
	if err != nil {
		c.HTML(500, "error.html", V{
			Error: err,
		})
		return
	}

	b.CoverPath = strings.TrimPrefix(b.CoverPath, app.bookDir)

	template := "detail.html"
	if c.Request.Header.Get("HX-Request") == "true" {
		template = "bookdetail"
	}

	c.HTML(200, template, V{
		Results:    0,
		Book:       b,
		ExtraPaths: books,
		IsAdmin:    c.GetBool("isAdmin"),
		TotalBooks: app.searchDB.GetBookCount(),
		Indexing:   app.state == "indexing",
	})

}
