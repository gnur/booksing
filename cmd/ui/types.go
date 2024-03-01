package main

import (
	"time"

	"github.com/gnur/booksing"
	"github.com/sirupsen/logrus"
)

// booksingApp holds all relevant global stuff for the booksing server
type booksingApp struct {
	db        database
	searchDB  searchDB
	bookDir   string
	importDir string
	logger    *logrus.Entry
	timezone  *time.Location
	adminUser string
	cfg       configuration
	state     string
}

type database interface {
	AddDownload(booksing.Download) error
	GetDownloads(int) ([]booksing.Download, error)

	SaveUser(*booksing.User) error
	GetUser(string) (booksing.User, error)

	GetUsers() ([]booksing.User, error)

	Close()
}

type searchDB interface {
	AddBooks([]booksing.Book) error
	GetBookCount() int
	HasHash(string) (bool, error)
	DeleteBook(string) error
	GetBooks(string, int64, int64) (*booksing.SearchResult, error)
	GetBook(string) (*booksing.Book, error)
}
