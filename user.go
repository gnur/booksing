package booksing

import (
	"time"
)

// User demo
type User struct {
	Username  string
	IsAdmin   bool
	IsAllowed bool
	Created   time.Time
	LastSeen  time.Time
	APIKeys   []Apikey
}
