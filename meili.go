package main

import (
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

type meiliDB struct {
	db    *meilisearch.Client
	index *meilisearch.Index
}

var stopWords = []string{"de", "het", "een", "the", "a", "an", "of", "and", "or", "in", "to", "for", "on", "at", "by"}

func NewMeiliSearch(host, key, indexName string) (*meiliDB, error) {

	slog.Info("Creating meili search client", "host", host)
	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host: host,
		//APIKey: key,
	})

	slog.Info("Creating meili search index", "index", indexName)
	index := client.Index(indexName)
	state, err := client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        indexName,
		PrimaryKey: "Hash",
	})
	if err != nil {
		slog.Warn("Failed to create index", "err", err)
		return nil, err
	}

	for {
		slog.Info("Waiting for index to be created")
		t, err := client.GetTask(state.TaskUID)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve meili task status: %w", err)
		}
		if t.Status == meilisearch.TaskStatusSucceeded {
			break
		}
		slog.Info("Index not ready yet", "status", t.Status)
		time.Sleep(10 * time.Millisecond)
	}

	cur, err := index.GetStopWords()
	if err != nil {
		slog.Warn("Failed to get stopWords", "err", err)
		return nil, err
	}

	slices.Sort(stopWords)

	if slices.Equal(*cur, stopWords) {
		slog.Info("StopWords are already set")
	} else {
		slog.Info("updating stopWords in database", "current", cur, "new", stopWords)
		_, err = index.UpdateStopWords(&stopWords)
		if err != nil {
			slog.Warn("Failed to update stopWords", "err", err)
			return nil, err
		}
	}

	return &meiliDB{
		db:    client,
		index: index,
	}, nil
}

func (db *meiliDB) GetBookCount() int {
	stats, err := db.index.GetStats()
	if err != nil {
		return 0
	}
	return int(stats.NumberOfDocuments)
}

func (db *meiliDB) HasHash(h string) (bool, error) {
	var doc Book
	err := db.index.GetDocument(h, nil, &doc)
	if doc.Hash == h {
		return true, nil
	}
	return false, err
}

func (db *meiliDB) GetBook(h string) (*Book, error) {
	var b Book
	err := db.index.GetDocument(h, nil, &b)
	return &b, err
}

func (db *meiliDB) AddBooks(books []Book) error {
	//TODO: do something with task info or ignore?
	_, err := db.index.AddDocuments(books)
	return err
}

func (db *meiliDB) DeleteBook(hash string) error {
	//TODO: do something with task info or ignore?
	_, err := db.index.DeleteDocument(hash)
	return err
}

func (db *meiliDB) GetBooks(q string, limit, offset int64) (*SearchResult, error) {

	var books []Book

	resp, err := db.index.Search(q, &meilisearch.SearchRequest{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	for _, hit := range resp.Hits {
		book, err := parseResult(hit)
		if err != nil {
			slog.Warn("Failed to decode book", "err", err)
		}
		books = append(books, *book)
	}

	return &SearchResult{
		Items: books,
		Total: resp.EstimatedTotalHits,
	}, nil
}
