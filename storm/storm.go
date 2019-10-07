package storm

import (
	"strings"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"github.com/gnur/booksing"
	log "github.com/sirupsen/logrus"
)

type stormDB struct {
	db *storm.DB
}

type Book = booksing.Book
type download = booksing.Download
type RefreshResult = booksing.RefreshResult

func New(path string) (*stormDB, error) {
	db, err := storm.Open(path)
	if err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"path": path,
		}).Error("could not open filedb")
	}

	database := stormDB{
		db: db,
	}

	return &database, nil
}

func (db *stormDB) Close() {
	db.db.Close()
}

func (db *stormDB) AddBook(b *Book) error {
	return db.db.Save(b)
}

func (db *stormDB) GetBook(query string) (*Book, error) {
	results, err := db.filterBooksBQL(query, 10)
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

func (db *stormDB) DeleteBook(hash string) error {
	return nil
}

func (db *stormDB) SetBookConverted(hash string) error {
	return nil
}

func (db *stormDB) GetBooks(query string, limit int) ([]Book, error) {
	if query == "" {
		return db.getRecentBooks(limit)
	}
	if strings.Contains(query, ":") {
		return db.filterBooksBQL(query, limit)
	}
	return nil, nil
}

func (db *stormDB) searchMetaphoneKeys(query string, limit int) ([]Book, error) {
	return nil, nil
}

func (db *stormDB) searchExact(query string, limit int) ([]Book, error) {
	return nil, nil
}

func (db *stormDB) filterBooksBQL(query string, limit int) ([]Book, error) {
	bql := parseQueryStorm(query)
	var books []Book
	err := db.db.Select(bql).Limit(limit).OrderBy("Author", "Title").Find(&books)
	return books, err
}

func (db *stormDB) getRecentBooks(limit int) ([]Book, error) {
	var books []Book
	err := db.db.Select(q.Eq("Language", "nl")).Limit(limit).Reverse().OrderBy("Added").Find(&books)
	return books, err
}

func (db *stormDB) AddDownload(dl download) error {
	return db.db.Save(&dl)
}
func (db *stormDB) GetBookBy(field, value string) (*Book, error) {
	//TODO: actually implement this
	return nil, nil
}
func (db *stormDB) GetDownloads(limit int) ([]download, error) {
	var dls []download
	err := db.db.All(&dls)
	return dls, err
}
func (db *stormDB) BookCount() int {
	return 0
}

func (db *stormDB) AddRefresh(rr RefreshResult) error {
	return db.db.Save(&rr)
}
func (db *stormDB) GetRefreshes(limit int) ([]RefreshResult, error) {
	var refreshes []RefreshResult
	err := db.db.All(&refreshes)
	return refreshes, err
}

func parseQueryStorm(s string) q.Matcher {
	first := true
	query := q.True()
	params := strings.Split(s, ",")
	for _, param := range params {
		parts := strings.Split(param, ":")
		if len(parts) != 2 {
			continue
		}

		field := strings.Title(strings.TrimSpace(parts[0]))
		filter := strings.TrimSpace(parts[1])
		log.WithFields(log.Fields{
			"field":  field,
			"filter": filter,
		}).Debug("creating filter")
		if first {
			query = q.Re(field, "(?i)"+filter)
			first = false
		} else {
			query = q.And(query, q.Eq(field, filter))
		}

	}

	return query
}
