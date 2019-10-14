package booksing

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"time"

	"strings"

	"github.com/gnur/booksing/epub"
	"github.com/kennygrant/sanitize"
	minio "github.com/minio/minio-go"
)

var yearRemove = regexp.MustCompile(`\((1|2)[0-9]{3}\)`)
var drukRemove = regexp.MustCompile(`(?i)/ druk [0-9]+`)
var filenameSafe = regexp.MustCompile("[^a-zA-Z0-9 -]+")

type StorageLocation string

const (
	S3Storage   StorageLocation = "S3"
	FileStorage StorageLocation = "FILE"
)

// Book represents a book record in the database, regular "book" data with extra metadata
type Book struct {
	ID            int                 `json:"stormid" storm:"id,increment"`
	Hash          string              `json:"hash" storm:"index"`
	Title         string              `json:"title" storm:"index"`
	Author        string              `json:"author" storm:"index"`
	Language      string              `json:"language" storm:"index"`
	Description   string              `json:"description"`
	MetaphoneKeys []string            `bson:"metaphone_keys"`
	SearchWords   []string            `bson:"search_keys"`
	Added         time.Time           `bson:"date_added" json:"date_added" storm:"index"`
	Locations     map[string]Location `json:"locations"`
}

// Location represents a storage location of a book
type Location struct {
	Type StorageLocation
	S3   *S3Location   `json:",omitempty"`
	File *FileLocation `json:",omitempty"`
}

type S3Location struct {
	Host   string
	Bucket string
	Key    string
}

func (s *S3Location) GetDLLink() (string, error) {
	accessKeyID := os.Getenv("ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("SECRET_ACCESS_KEY")

	mc, err := minio.New(s.Host, accessKeyID, secretAccessKey, true)

	if err != nil {
		return "", err
	}
	url, err := mc.PresignedGetObject(s.Bucket, s.Key, 12*time.Hour, nil)

	if err != nil {
		return "", err
	}
	return url.String(), nil
}

type FileLocation struct {
	Path string
}

func (b *Book) HasSearchWords(terms []string) bool {
	for _, term := range terms {
		if !contains(b.SearchWords, term) {
			return false
		}
	}
	return true
}

func (b *Book) HasMetaphoneKeys(terms []string) bool {
	for _, term := range terms {
		if !contains(b.MetaphoneKeys, term) {
			return false
		}
	}
	return true
}

func (b *Book) HasMobi() bool {
	_, exists := b.Locations["mobi"]
	return exists
}

// NewBookFromFile creates a book object from a file
func NewBookFromFile(bookpath string, rename bool, baseDir string) (bk *Book, err error) {
	epub, err := epub.ParseFile(bookpath)
	if err != nil {
		return nil, err
	}

	book := Book{
		Title:       epub.Title,
		Author:      epub.Author,
		Language:    epub.Language,
		Description: epub.Description,
	}

	f, err := os.Open(bookpath)
	if err != nil {
		return nil, err
	}

	mobiPath := strings.Replace(bookpath, "epub", "mobi", -1)
	_, err = os.Stat(mobiPath)
	if !os.IsNotExist(err) {
		book.Locations["mobi"] = Location{
			Type: FileStorage,
			File: &FileLocation{
				Path: mobiPath,
			},
		}
	}

	fp := bookpath

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	book.Added = fi.ModTime()

	book.Title = Fix(book.Title, true, false)
	book.Author = Fix(book.Author, true, true)
	book.Language = FixLang(book.Language)
	book.Description = sanitize.HTML(book.Description)

	searchWords := book.Title + " " + book.Author
	book.MetaphoneKeys = GetMetaphoneKeys(searchWords)
	book.SearchWords = GetLowercasedSlice(searchWords)

	book.Hash = HashBook(book.Author, book.Title)

	if rename {
		newBookPath := path.Join(baseDir, GetBookPath(book.Author, book.Title)+".epub")
		if bookpath != newBookPath {
			baseDir := filepath.Dir(newBookPath)
			err := os.MkdirAll(baseDir, 0755)
			if err == nil {
				os.Rename(bookpath, newBookPath)
				fp = newBookPath
			}
		}
	}
	book.Locations = make(map[string]Location)
	book.Locations["epub"] = Location{
		Type: FileStorage,
		File: &FileLocation{
			Path: fp,
		},
	}

	return &book, nil
}

func GetBookPath(title, author string) string {
	author = filenameSafe.ReplaceAllString(author, "")
	title = filenameSafe.ReplaceAllString(title, "")
	if len(title) > 35 {
		title = title[:30]
	}
	title = strings.TrimSpace(title)
	author = strings.TrimSpace(author)
	if len(author) == 0 {
		author = "unknown"
	}
	if len(title) == 0 {
		author = "unknown"
	}
	parts := strings.Split(author, " ")
	firstChar := parts[len(parts)-1][0:1]
	formatted := fmt.Sprintf("%s/%s/%s-%s", firstChar, author, author, title)
	formatted = strings.Replace(formatted, " ", "_", -1)
	formatted = strings.Replace(formatted, "__", "_", -1)

	return formatted
}

func FixLang(s string) string {
	s = strings.ToLower(s)

	switch s {
	case "nld":
		s = "nl"
	case "dutch":
		s = "nl"
	case "nederlands":
		s = "nl"
	case "nederland":
		s = "nl"
	case "nl-nl":
		s = "nl"
	case "nl_nl":
		s = "nl"
	case "dut":
		s = "nl"

	case "deutsch":
		s = "de"
	case "deutsche":
		s = "de"
	case "duits":
		s = "de"
	case "german":
		s = "de"
	case "ger":
		s = "de"
	case "de-de":
		s = "de"
	case "de_de":
		s = "de"

	case "english":
		s = "en"
	case "engels":
		s = "en"
	case "eng":
		s = "en"
	case "uk":
		s = "en"
	case "en-us":
		s = "en"
	case "en-gb":
		s = "en"
	case "en-en":
		s = "en"
	case "en_us":
		s = "en"
	case "en_gb":
		s = "en"
	case "en_en":
		s = "en"
	case "us":
		s = "en"
	}
	return s
}

func Fix(s string, capitalize, correctOrder bool) string {
	if s == "" {
		return "Unknown"
	}
	if capitalize {
		s = strings.Title(strings.ToLower(s))
		s = strings.Replace(s, "'S", "'s", -1)
	}
	if correctOrder && strings.Contains(s, ",") {
		sParts := strings.Split(s, ",")
		if len(sParts) == 2 {
			s = strings.TrimSpace(sParts[1]) + " " + strings.TrimSpace(sParts[0])
		}
	}

	s = yearRemove.ReplaceAllString(s, "")
	s = drukRemove.ReplaceAllString(s, "")
	s = strings.Replace(s, ".", " ", -1)
	s = strings.Replace(s, "  ", " ", -1)
	s = strings.TrimSpace(s)

	return strings.Map(func(in rune) rune {
		switch in {
		case '“', '‹', '”', '›':
			return '"'
		case '_':
			return ' '
		case '‘', '’':
			return '\''
		}
		return in
	}, s)
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
