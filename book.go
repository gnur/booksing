package booksing

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"time"

	"strings"

	"github.com/gnur/booksing/epub"
	"github.com/kennygrant/sanitize"
)

var yearRemove = regexp.MustCompile(`\((1|2)[0-9]{3}\)`)
var drukRemove = regexp.MustCompile(`(?i)/ druk [0-9]+`)
var filenameSafe = regexp.MustCompile("[^a-zA-Z0-9 -]+")

type StorageLocation string

var ErrFileAlreadyExists = errors.New("Target file already exists")
var ErrCoverWriteFailed = errors.New("Failed to write cover")

const (
	FileStorage StorageLocation = "FILE"
)

// Book represents a book record in the database, regular "book" data with extra metadata
type Book struct {
	Hash        string
	Title       string
	Author      string
	Language    string
	Description string
	Added       time.Time
	Path        string
	Size        int64
	HasCover    bool
	CoverPath   string
	Publisher   string
	ISBN        string
	Series      string
	PublishDate time.Time
	SeriesIndex float64
}

type BookInput struct {
	Title       string
	Author      string
	Language    string
	Description string
	Path        string
}

func (b *BookInput) ToBook() Book {
	var book Book
	book.Author = Fix(b.Author, true, true)
	book.Title = Fix(b.Title, true, false)
	book.Language = FixLang(b.Language)
	book.Description = b.Description
	book.Path = b.Path

	book.Hash = HashBook(book.Author, book.Title)

	return book

}

type FileLocation struct {
	Path string
}

// NewBookFromFile creates a book object from a file
func NewBookFromFile(bookpath string, baseDir string) (bk *Book, err error) {
	epub, cover, err := epub.ParseFile(bookpath)
	if err != nil {
		fmt.Println(cover)
		return nil, err
	}

	book := Book{
		Title:       epub.Title,
		Author:      epub.Author,
		Language:    epub.Language,
		Description: epub.Description,
		HasCover:    epub.HasCover,
		Publisher:   epub.Publisher,
		ISBN:        epub.ISBN,
		Series:      epub.Series,
		PublishDate: epub.PublishDate,
		SeriesIndex: epub.SeriesIndex,
	}

	f, err := os.Open(bookpath)
	if err != nil {
		return nil, err
	}

	fp := bookpath

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	book.Added = fi.ModTime()
	book.Size = fi.Size()

	book.Title = Fix(book.Title, true, false)
	book.Author = Fix(book.Author, true, true)
	book.Language = FixLang(book.Language)
	book.Description = sanitize.HTML(book.Description)

	book.Hash = HashBook(book.Author, book.Title)

	newBookPath := path.Join(baseDir, GetBookPath(book.Title, book.Author)+".epub")
	if _, err := os.Stat(newBookPath); err == nil {
		return &book, ErrFileAlreadyExists
	}
	baseDir = filepath.Dir(newBookPath)
	err = os.MkdirAll(baseDir, 0755)
	if err == nil {
		_ = os.Rename(bookpath, newBookPath)
		fp = newBookPath
	}
	book.Path = fp
	if book.HasCover {
		book.CoverPath = strings.Replace(fp, ".epub", ".jpg", 1)
		err = ioutil.WriteFile(book.CoverPath, cover, 0644)
		if err != nil {
			return &book, ErrCoverWriteFailed
		}
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
