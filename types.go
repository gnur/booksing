package booksing

import (
	"errors"
)

var ErrNonUniqueResult = errors.New("query gave more then 1 result")
var ErrNotFound = errors.New("query no results")
var ErrDuplicate = errors.New("duplicate key")

type SearchResult struct {
	Items []Book
	Total int64
}
