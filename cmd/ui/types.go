package main

import (
	"time"

	"github.com/gnur/booksing"
)

// booksingApp holds all relevant global stuff for the booksing server
type booksingApp struct {
	searchDB  searchDB
	bookDir   string
	importDir string
	timezone  *time.Location
	adminUser string
	cfg       configuration
	state     string
}

type searchDB interface {
	AddBooks([]booksing.Book) error
	GetBookCount() int
	HasHash(string) (bool, error)
	DeleteBook(string) error
	GetBooks(string, int64, int64) (*booksing.SearchResult, error)
	GetBook(string) (*booksing.Book, error)
}
