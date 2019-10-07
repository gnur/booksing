package main

import (
	"time"

	"github.com/gnur/booksing"
)

type booksingApp struct {
	db            database
	allowDeletes  bool
	allowOrganize bool
	bookDir       string
	importDir     string
}

type download = booksing.Download
type RefreshResult = booksing.RefreshResult

type bookResponse struct {
	Books      []booksing.Book `json:"books"`
	TotalCount int             `json:"total"`
	timestamp  time.Time
}

type parseResult int32

// hold all possible book parse results
const (
	OldBook       parseResult = iota
	AddedBook     parseResult = iota
	DuplicateBook parseResult = iota
	InvalidBook   parseResult = iota
)

type database interface {
	AddBook(*booksing.Book) error
	BookCount() int
	GetBook(string) (*booksing.Book, error)
	DeleteBook(string) error
	GetBooks(string, int) ([]booksing.Book, error)
	SetBookConverted(string) error

	GetBookBy(string, string) (*booksing.Book, error)

	AddDownload(booksing.Download) error
	GetDownloads(int) ([]booksing.Download, error)

	AddRefresh(RefreshResult) error
	GetRefreshes(int) ([]RefreshResult, error)
	Close()
}
