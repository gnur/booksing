package main

import (
	"errors"
	"time"
)

var ErrNonUniqueResult = errors.New("query gave more then 1 result")
var ErrNotFound = errors.New("query no results")
var ErrDuplicate = errors.New("duplicate key")

type SearchResult struct {
	Items []Book
	Total int64
}

// booksingApp holds all relevant global stuff for the booksing server
type booksingApp struct {
	searchDB       searchDB
	bookDir        string
	importDir      string
	timezone       *time.Location
	cfg            configuration
	state          string
	webHookEnabled bool
}

type searchDB interface {
	AddBooks([]Book) error
	GetBookCount() int
	HasHash(string) (bool, error)
	DeleteBook(string) error
	GetBooks(string, int64, int64) (*SearchResult, error)
	GetBook(string) (*Book, error)
}
