package booksing

import (
	"time"
)

type Apikey struct {
	ID       string `json:"id"`
	Username string
	Key      string
	Created  time.Time
	LastUsed time.Time
}
