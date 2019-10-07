package firestore

import (
	"context"
	"fmt"
	"strings"

	
	"cloud.google.com/go/firestore"
	"github.com/gnur/booksing"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
)

// FireDB holds the firestore client
type FireDB struct {
	client *firestore.Client
}

// NewFireStore returns a new firestore client
func New(projectID string) (*FireDB, error) {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	db := FireDB{
		client: client,
	}
	return &db, nil
}
func (db *FireDB) Close() {
	db.client.Close()
}

func (db *FireDB) AddBook(b *booksing.Book) error {
	ctx := context.Background()

	_, err := db.GetBookBy("Hash", b.Hash)
	if err == nil {
		return booksing.ErrDuplicate
	}
	_, err = db.client.Collection("books").Doc(b.Hash).Set(ctx, b)

	return err
}

func (db *FireDB) GetBook(q string) (*booksing.Book, error) {
	results, err := db.filterBooksBQL(q, 10)
	if err != nil {
		return nil, err
	}
	if len(results) > 1 {
		return nil, booksing.ErrNonUniqueResult
	}
	if len(results) == 0 {
		return nil, booksing.ErrNotFound
	}
	return &results[0], nil
}

func (db *FireDB) GetBookBy(field, value string) (*booksing.Book, error) {
	ctx := context.Background()
	iter := db.client.Collection("books").Where(field, "==", value).Limit(5).Documents(ctx)

	var books []booksing.Book
	var b booksing.Book
	for {
		b = booksing.Book{}
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to iterate: %v", err)
		}
		err = doc.DataTo(&b)
		if err == nil {
			books = append(books, b)
		}
	}

	if len(books) > 1 {
		return nil, booksing.ErrNonUniqueResult
	}
	if len(books) == 0 {
		return nil, booksing.ErrNotFound
	}

	return &books[0], nil
}

func (db *FireDB) DeleteBook(hash string) error {
	ctx := context.Background()
	_, err := db.client.Collection("books").Doc(hash).Delete(ctx)
	return err
}

func (db *FireDB) SetBookConverted(hash string) error {
	ctx := context.Background()
	book, _ := db.GetBookBy("Hash", hash)
	book.HasMobi = true

	_, err := db.client.Collection("books").Doc(hash).Set(ctx, book)

	return err
}

func (db *FireDB) GetBooks(q string, limit int) ([]booksing.Book, error) {
	if q == "" {
		return db.getRecentBooks(limit)
	}
	if strings.Contains(q, ":") {
		return db.filterBooksBQL(q, limit)
	}

	books, err := db.searchExact(q, limit)
	if err != nil {
		log.WithField("err", err).Error("filtering books failed")
		return []booksing.Book{}, err
	}
	if len(books) > 0 {
		return books, nil
	}

	books, err = db.searchMetaphoneKeys(q, limit)
	if err != nil {
		log.WithField("err", err).Error("filtering books failed")
		return []booksing.Book{}, err
	}
	return books, nil
}

func (db *FireDB) searchMetaphoneKeys(q string, limit int) ([]booksing.Book, error) {
	ctx := context.Background()

	longestTermLength := 0
	longestTerm := ""

	terms := booksing.GetMetaphoneKeys(q)

	for _, term := range terms {
		if len(term) > longestTermLength {
			longestTerm = term
			longestTermLength = len(term)
		}
	}

	iter := db.client.Collection("books").Where("MetaphoneKeys", "array-contains", longestTerm).Limit(limit).Documents(ctx)

	books, err := iterToBookList(iter)
	if err != nil {
		return nil, err
	}

	var retBooks []booksing.Book
	for _, book := range books {
		if book.HasMetaphoneKeys(terms) {
			retBooks = append(retBooks, book)
		}
	}
	return retBooks, nil
}

func (db *FireDB) searchExact(q string, limit int) ([]booksing.Book, error) {
	ctx := context.Background()

	longestTermLength := 0
	longestTerm := ""

	terms := strings.Split(q, " ")

	for _, term := range terms {
		if len(term) > longestTermLength {
			longestTerm = term
			longestTermLength = len(term)
		}
	}

	iter := db.client.Collection("books").Where("SearchWords", "array-contains", longestTerm).Limit(limit).Documents(ctx)

	books, err := iterToBookList(iter)
	if err != nil {
		return nil, err
	}

	var retBooks []booksing.Book
	for _, book := range books {
		if book.HasSearchWords(terms) {
			retBooks = append(retBooks, book)
		}
	}
	return retBooks, nil
}

func (db *FireDB) filterBooksBQL(q string, limit int) ([]booksing.Book, error) {
	query := db.parseQuery(q)
	var books []booksing.Book
	var b booksing.Book

	ctx := context.Background()
	iter := query.OrderBy("Hash", firestore.Desc).Limit(limit).Documents(ctx)
	for {
		b = booksing.Book{}
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to iterate: %v", err)
		}
		err = doc.DataTo(&b)
		if err == nil {
			books = append(books, b)
		}
	}

	return books, nil
}

func (db *FireDB) getRecentBooks(limit int) ([]booksing.Book, error) {
	var books []booksing.Book
	ctx := context.Background()
	var b booksing.Book
	iter := db.client.Collection("books").OrderBy("Added", firestore.Desc).Limit(limit).Documents(ctx)
	for {
		b = booksing.Book{}
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to iterate: %v", err)
		}
		err = doc.DataTo(&b)
		if err == nil {
			books = append(books, b)
		}
	}
	return books, nil
}

func (db *FireDB) AddDownload(dl booksing.Download) error {
	ctx := context.Background()
	_, _, err := db.client.Collection("downloads").Add(ctx, dl)
	return err
}
func (db *FireDB) GetDownloads(limit int) ([]booksing.Download, error) {
	var dls []booksing.Download
	ctx := context.Background()
	var d booksing.Download
	iter := db.client.Collection("downloads").OrderBy("Timestamp", firestore.Desc).Limit(limit).Documents(ctx)
	for {
		d = booksing.Download{}
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to iterate: %v", err)
		}
		err = doc.DataTo(&d)
		if err == nil {
			dls = append(dls, d)
		}
	}
	return dls, nil
}
func (db *FireDB) BookCount() int {
	return 0
}

func (db *FireDB) AddRefresh(rr booksing.RefreshResult) error {
	ctx := context.Background()
	_, _, err := db.client.Collection("refreshes").Add(ctx, rr)
	return err
}
func (db *FireDB) GetRefreshes(limit int) ([]booksing.RefreshResult, error) {
	var refreshes []booksing.RefreshResult
	ctx := context.Background()
	var r booksing.RefreshResult
	iter := db.client.Collection("refreshes").OrderBy("StartTime", firestore.Desc).Limit(limit).Documents(ctx)
	for {
		r = booksing.RefreshResult{}
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to iterate: %v", err)
		}
		err = doc.DataTo(&r)
		if err == nil {
			refreshes = append(refreshes, r)
		}
	}

	return refreshes, nil
}

func (db *FireDB) parseQuery(s string) firestore.Query {
	col := db.client.Collection("books")
	var q firestore.Query
	params := strings.Split(s, ",")
	first := true
	for _, param := range params {
		parts := strings.Split(param, ":")
		if len(parts) != 2 {
			continue
		}

		field := strings.TrimSpace(parts[0])
		filter := strings.TrimSpace(parts[1])

		if first {
			q = col.Where(field, "==", filter)
		} else {
			q = q.Where(field, "==", filter)
		}
	}

	return q
}

func iterToBookList(iter *firestore.DocumentIterator) ([]booksing.Book, error) {
	var books []booksing.Book
	var b booksing.Book
	for {
		b = booksing.Book{}
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to iterate: %v", err)
		}
		err = doc.DataTo(&b)
		if err == nil {
			books = append(books, b)
		}
	}
	return books, nil
}
