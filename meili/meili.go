package meili

import (
	"log/slog"
	"slices"

	"github.com/gnur/booksing"
	"github.com/meilisearch/meilisearch-go"
)

type meiliDB struct {
	db    *meilisearch.Client
	index *meilisearch.Index
}

var stopWords = []string{"de", "het", "een", "the", "a", "an", "of", "and", "or", "in", "to", "for", "on", "at", "by"}

func New(host, key, indexName string) (*meiliDB, error) {

	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host: host,
		//APIKey: key,
	})
	// An index is where the documents are stored.
	index := client.Index(indexName)
	_, err := client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        indexName,
		PrimaryKey: "Hash",
	})
	if err != nil {
		slog.Warn("Failed to create index", "err", err)
		return nil, err
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
	var doc booksing.Book
	err := db.index.GetDocument(h, nil, &doc)
	if doc.Hash == h {
		return true, nil
	}
	return false, err
}

func (db *meiliDB) GetBook(h string) (*booksing.Book, error) {
	var b booksing.Book
	err := db.index.GetDocument(h, nil, &b)
	return &b, err
}

func (db *meiliDB) AddBooks(books []booksing.Book) error {
	//TODO: do something with task info or ignore?
	_, err := db.index.AddDocuments(books)
	return err
}

func (db *meiliDB) DeleteBook(hash string) error {
	//TODO: do something with task info or ignore?
	_, err := db.index.DeleteDocument(hash)
	return err
}

func (db *meiliDB) GetBooks(q string, limit, offset int64) (*booksing.SearchResult, error) {

	var books []booksing.Book

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

	return &booksing.SearchResult{
		Items: books,
		Total: resp.EstimatedTotalHits,
	}, nil
}
