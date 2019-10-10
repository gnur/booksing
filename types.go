package booksing

import (
	"errors"
	"time"
)

var ErrNonUniqueResult = errors.New("Query gave more then 1 result")
var ErrNotFound = errors.New("Query no results")
var ErrDuplicate = errors.New("Duplicate key")

// RefreshResult holds the result of a full refresh
type RefreshResult struct {
	ID        int `storm:"id,increment"`
	StartTime time.Time
	StopTime  time.Time
	Old       int
	Added     int
	Duplicate int
	Invalid   int
}

type Download struct {
	ID        int       `storm:"id,increment"`
	Book      string    `json:"hash"`
	User      string    `json:"user"`
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
}

type PipelineResult struct {
	Title  string   `bson:"_id"`
	Count  int      `bson:"count"`
	Hashes []string `bson:"docs"`
}

type AddBookInput struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	Language    string `json:"language"`
	Description string `json:"description"`
}
