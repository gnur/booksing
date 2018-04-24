package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/asdine/storm"
	"github.com/asdine/storm/codec/msgpack"
)

func getDB() *storm.DB {
	bookDir := os.Getenv("BOOK_DIR")
	if bookDir == "" {
		bookDir = "."
	}
	dbLocation := os.Getenv("DATABASE_LOCATION")
	if dbLocation == "" {
		dbLocation = filepath.Join(bookDir, "booksing.db")
	}
	db, err := storm.Open(dbLocation, storm.Codec(msgpack.Codec), storm.Batch())
	if err != nil {
		fmt.Println(err)
	}
	return db
}

func benchmarkFilter(db *storm.DB, filter string, limit int, b *testing.B) {
	for n := 0; n < b.N; n++ {
		_ = filterBooksFast(db, filter, limit)
	}
}

func BenchmarkFilterEmpty50(b *testing.B) {
	db := getDB()
	benchmarkFilter(db, "", 50, b)
}

func BenchmarkFilterNo50(b *testing.B) {
	db := getDB()
	benchmarkFilter(db, "asldkfjalskdjf", 50, b)
}
func BenchmarkFilterWith50(b *testing.B) {
	db := getDB()
	benchmarkFilter(db, "george", 50, b)
}
func BenchmarkFilterEmpty200(b *testing.B) {
	db := getDB()
	benchmarkFilter(db, "", 200, b)
}

func BenchmarkFilterNo200(b *testing.B) {
	db := getDB()
	benchmarkFilter(db, "asldkfjalskdjf", 200, b)
}
func BenchmarkFilterWith200(b *testing.B) {
	db := getDB()
	benchmarkFilter(db, "george", 200, b)
}
