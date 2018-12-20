package main

import (
	"time"
)

type bookResponse struct {
	Books      []Book `json:"books"`
	TotalCount int    `json:"total"`
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

// RefreshResult holds the result of a full refresh
type RefreshResult struct {
	StartTime time.Time
	StopTime  time.Time
	Old       int
	Added     int
	Duplicate int
	Invalid   int
}

type download struct {
	Book      string    `json:"hash"`
	User      string    `json:"user"`
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
}

type bookConvertRequest struct {
	Hash          string `json:"bookhash"`
	Receiver      string `json:"email"`
	SMTPServer    string `json:"smtpserver"`
	SMTPUser      string `json:"smtpuser"`
	SMTPPassword  string `json:"smtppass"`
	ConvertToMobi bool   `json:"convert"`
}

type pipelineResult struct {
	Title  string   `bson:"_id"`
	Count  int      `bson:"count"`
	Hashes []string `bson:"docs"`
}