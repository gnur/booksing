package main

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gnur/booksing"
	"github.com/sirupsen/logrus"
)

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

	books, err := app.db.GetBooks(q, limit, offset)
	if err != nil {
		c.HTML(500, "error.html", V{
			Error: err,
			Q:     q,
		})
		return
	}

	u := c.MustGet("id")
	user := u.(*booksing.User)
	username := user.Name

	for i, b := range books.Items {
		icon, err := app.getUserIcon(username, b.Hash)
		app.logger.WithFields(logrus.Fields{
			"icon": icon,
			"err":  err,
		}).Info("checking bm")
		b := &books.Items[i]
		b.Icon, _ = app.getUserIcon(username, b.Hash)
		app.logger.WithFields(logrus.Fields{
			"icon": books.Items[i].Icon,
		}).Info("Result in slice")
	}

	stop := time.Since(start)
	latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))
	c.HTML(200, "search.html", V{
		Limit:      limit,
		Offset:     offset,
		Results:    books.Total,
		TimeTaken:  latency,
		Books:      books.Items,
		Error:      err,
		Q:          q,
		IsAdmin:    c.GetBool("isAdmin"),
		TotalBooks: app.db.GetBookCount(),
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
		TotalBooks: app.db.GetBookCount(),
		Users:      users,
		Indexing:   app.state == "indexing",
	})

}

func (app *booksingApp) showStats(c *gin.Context) {
	start := time.Now().Add(-365 * 24 * time.Hour)
	end := time.Now()

	stats, err := app.db.GetBookCountHistory(start, end)
	if err != nil {
		c.HTML(403, "error.html", V{
			Error: err,
		})
		c.Abort()
		return
	}

	c.HTML(200, "stats.html", V{
		Error:      err,
		Q:          "",
		IsAdmin:    c.GetBool("isAdmin"),
		TotalBooks: app.db.GetBookCount(),
		Stats:      stats,
		Indexing:   app.state == "indexing",
	})

}

func (app *booksingApp) deleteBook(c *gin.Context) {
	hash := c.Param("hash")

	book, err := app.db.GetBook(hash)
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

	err = app.db.DeleteBook(hash)
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
	err = app.db.UpdateBookCount(-1)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"hash": hash,
			"err":  err,
		}).Error("could not update book count")
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
		TotalBooks: app.db.GetBookCount(),
		Downloads:  dls,
		Indexing:   app.state == "indexing",
	})

}

func (app *booksingApp) rotateIcon(c *gin.Context) {
	hash := c.Param("hash")

	u := c.MustGet("id")
	user := u.(*booksing.User)

	currentIcon := booksing.DefaultShelveIcon()

	bm, ok := user.Bookmarks[hash]
	if ok {
		currentIcon = bm.Icon
	}

	newIcon, err := booksing.NextShelveIcon(currentIcon)
	if err != nil {
		newIcon = booksing.DefaultShelveIcon()
	}

	user.Bookmarks[hash] = booksing.Bookmark{
		Icon:       newIcon,
		LastChange: time.Now(),
	}

	if newIcon == booksing.DefaultShelveIcon() {
		delete(user.Bookmarks, hash)
	}

	err = app.db.SaveUser(user)
	if err != nil {
		c.HTML(500, "error.html", V{
			Error: err,
		})
		return
	}
	if c.Query("method") == "manual" {
		c.Redirect(302, c.Request.Referer())
		return
	}
	c.JSON(200, gin.H{
		"new": newIcon,
		"msg": "ok",
	})

}

func (app *booksingApp) bookmarks(c *gin.Context) {
	u := c.MustGet("id")
	user := u.(*booksing.User)
	var books []booksing.Book

	start := time.Now()

	for hash := range user.Bookmarks {
		b, err := app.db.GetBook(hash)
		if err != nil {
			continue
		}
		books = append(books, *b)
	}

	stop := time.Since(start)
	latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))
	c.HTML(200, "bookmarks.html", V{
		Results:    int64(len(books)),
		TimeTaken:  latency,
		Books:      books,
		Q:          "",
		IsAdmin:    c.GetBool("isAdmin"),
		TotalBooks: app.db.GetBookCount(),
		Indexing:   app.state == "indexing",
	})
}

func (app *booksingApp) serveIcon(c *gin.Context) {
	hash := c.Param("hash")

	u := c.MustGet("id")
	user := u.(*booksing.User)

	hash = hash[:len(hash)-4]

	currentIcon := booksing.DefaultShelveIcon()
	bm, ok := user.Bookmarks[hash]
	if ok {
		currentIcon = bm.Icon
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/static/%s.png", currentIcon))

}

func (app *booksingApp) getUserIcon(username, hash string) (icon booksing.ShelveIcon, err error) {
	icon = booksing.DefaultShelveIcon()

	user, err := app.db.GetUser(username)
	if err != nil {
		app.logger.WithError(err).Warning("unable to get user from db")
		return
	}

	bm, ok := user.Bookmarks[hash]
	if ok {
		icon = bm.Icon
	}

	return
}
