package mongodb

import (
	"strings"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/gnur/booksing"
	log "github.com/sirupsen/logrus"
)

type mongoDB struct {
	books          *mgo.Collection
	downloads      *mgo.Collection
	refreshResults *mgo.Collection
}

type download = booksing.Download
type RefreshResult = booksing.RefreshResult

func New(host string) (*mongoDB, error) {
	conn, err := mgo.Dial(host)
	if err != nil {
		log.WithField("err", err).Error("Could not connect to mongodb")
		return nil, err
	}
	session := conn.DB("booksing")
	if err != nil {
		log.WithField("err", err).Error("Could not create booksing session")
		return nil, err
	}

	database := mongoDB{
		books:          session.C("books"),
		downloads:      session.C("downloads"),
		refreshResults: session.C("refreshResults"),
	}

	err = database.createIndices()
	if err != nil {
		log.WithField("err", err).Error("Could not create required indices")
		return nil, err
	}

	return &database, nil
}
func (db *mongoDB) Close() {}

func (db *mongoDB) GetBookBy(field, value string) (*booksing.Book, error) {
	//TODO: actually implement this
	return nil, nil
}

func (db *mongoDB) AddBook(b *booksing.Book) error {
	err := db.books.Insert(b)

	if mgo.IsDup(err) {
		return booksing.ErrDuplicate
	}
	return err
}

func (db *mongoDB) GetBook(q string) (*booksing.Book, error) {
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

func (db *mongoDB) DeleteBook(hash string) error {
	return db.books.Remove(bson.M{"hash": hash})
}

func (db *mongoDB) SetBookConverted(hash string) error {
	book, _ := db.GetBookBy("Hash", hash)
	book.HasMobi = true

	return db.books.Update(bson.M{"hash": hash}, book)
}

func (db *mongoDB) GetBooks(q string, limit int) ([]booksing.Book, error) {
	if q == "" {
		return db.getRecentBooks(limit)
	}
	if strings.Contains(q, ":") {
		return db.filterBooksBQL(q, limit)
	}

	books, err := db.searchExact(q, limit)
	if err != nil {
		log.WithField("err", err).Error("filtering books failed")
		return nil, err
	}
	if len(books) > 0 {
		return books, nil
	}

	return db.searchMetaphoneKeys(q, limit)
}

func (db *mongoDB) searchMetaphoneKeys(q string, limit int) ([]booksing.Book, error) {
	var books []booksing.Book
	var iter *mgo.Iter
	s := booksing.GetMetaphoneKeys(q)
	iter = db.books.Find(bson.M{"metaphone_keys": bson.M{"$all": s}}).Limit(limit).Sort("author", "title").Iter()
	err := iter.All(&books)
	if err != nil {
		log.WithField("err", err).Error("filtering books failed")
		return nil, err
	}
	return books, nil
}

func (db *mongoDB) searchExact(q string, limit int) ([]booksing.Book, error) {
	var books []booksing.Book
	var iter *mgo.Iter
	s := strings.Split(q, " ")
	iter = db.books.Find(bson.M{"search_keys": bson.M{"$all": s}}).Limit(limit).Sort("author", "title").Iter()
	err := iter.All(&books)
	if err != nil {
		log.WithField("err", err).Error("filtering books failed")
		return nil, err
	}
	return books, nil
}

func (db *mongoDB) filterBooksBQL(q string, limit int) ([]booksing.Book, error) {
	bsonQ := parseQuery(q)
	iter := db.books.Find(bsonQ).Limit(limit).Sort("author", "title").Iter()

	var books []booksing.Book
	err := iter.All(&books)
	if err != nil {
		return nil, err
	}
	return books, nil
}

func (db *mongoDB) getRecentBooks(limit int) ([]booksing.Book, error) {
	iter := db.books.Find(bson.M{"language": "nl"}).Sort("-date_added").Limit(limit).Iter()
	var books []booksing.Book
	err := iter.All(&books)
	if err != nil {
		return nil, err
	}
	return books, nil
}

func (db *mongoDB) AddDownload(dl download) error {
	return db.downloads.Insert(dl)
}
func (db *mongoDB) GetDownloads(limit int) ([]download, error) {
	var dls []download

	err := db.downloads.Find(nil).Sort("-timestamp").Iter().All(&dls)
	return dls, err
}
func (db *mongoDB) BookCount() int {
	count, _ := db.books.Count()
	return count
}

func (db *mongoDB) AddRefresh(rr RefreshResult) error {
	return db.refreshResults.Insert(rr)
}
func (db *mongoDB) GetRefreshes(limit int) ([]RefreshResult, error) {
	var refreshes []RefreshResult

	err := db.refreshResults.Find(nil).Sort("-starttime").Iter().All(&refreshes)

	return refreshes, err
}

func (db *mongoDB) createIndices() error {
	indices := []mgo.Index{
		mgo.Index{
			Key:      []string{"hash"},
			Unique:   true,
			DropDups: true,
		},
		mgo.Index{
			Key:      []string{"filepath"},
			Unique:   true,
			DropDups: true,
		},
		mgo.Index{
			Key:      []string{"metaphone_keys"},
			Unique:   false,
			DropDups: false,
		},
		mgo.Index{
			Key:      []string{"search_keys"},
			Unique:   false,
			DropDups: false,
		},
		mgo.Index{
			Key:      []string{"author"},
			Unique:   false,
			DropDups: false,
		},
		mgo.Index{
			Key:      []string{"title"},
			Unique:   false,
			DropDups: false,
		},
		mgo.Index{
			Key:      []string{"date_added"},
			Unique:   false,
			DropDups: false,
		},
		mgo.Index{
			Key:      []string{"language"},
			Unique:   false,
			DropDups: false,
		},
	}
	for _, index := range indices {
		err := db.books.EnsureIndex(index)
		if err != nil {
			log.WithFields(log.Fields{
				"index": index.Key,
				"err":   err,
			}).Error("Could not create index")
			return err
		}
	}
	return nil
}

func parseQuery(s string) bson.M {
	q := bson.M{}
	params := strings.Split(s, ",")
	for _, param := range params {
		parts := strings.Split(param, ":")
		if len(parts) != 2 {
			continue
		}

		field := strings.TrimSpace(parts[0])
		filter := strings.TrimSpace(parts[1])

		q[field] = bson.M{
			"$regex":   filter,
			"$options": "i",
		}
	}

	return q
}
