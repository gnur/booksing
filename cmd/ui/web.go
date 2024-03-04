package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/gnur/booksing"
)

func static(w http.ResponseWriter, r *http.Request) {
	fsPublic, _ := fs.Sub(booksing.NuxtElements, "web/.output/public")
	fs := http.FileServer(http.FS(fsPublic))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	p := r.URL.Path
	if strings.HasSuffix(p, ".css") {
		w.Header().Set("Content-Type", "text/css")
	}
	if strings.HasSuffix(p, ".js") {
		slog.Info("serving js", "path", p)
		w.Header().Set("Content-Type", "application/javascript")
	}

	fs.ServeHTTP(w, r)
}

func index(w http.ResponseWriter, r *http.Request) {
	w.Write(booksing.NuxtIndexHTML)
}

func bookPNG(w http.ResponseWriter, r *http.Request) {
	w.Write(booksing.BookPNG)
}

func (app *booksingApp) getCover(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=86400, immutable")

	//join the path with a slash to make sure it is an absolute path
	//and the Join will also automatically clean out any path traversal characters
	file := path.Join("/", r.URL.Query().Get("file"))

	//join only with the bookDir after the first join so only files from the bookdir are served
	file = path.Join(app.bookDir, file)

	http.ServeFile(w, r, file)

}

func (app *booksingApp) searchAPI(w http.ResponseWriter, r *http.Request) {
	var offset int64
	var limit int64
	var err error
	offset = 0
	limit = 20
	q := r.URL.Query().Get("q")
	off := r.URL.Query().Get("o")
	if off != "" {
		offset, err = strconv.ParseInt(off, 10, 64)
		if err != nil {
			offset = 0
		}
	}
	lim := r.URL.Query().Get("l")
	if lim != "" {
		limit, err = strconv.ParseInt(lim, 10, 64)
		if err != nil {
			limit = 20
		}
	}

	var books *booksing.SearchResult

	books, err = app.searchDB.GetBooks(q, limit, offset)
	if err != nil {
		slog.Warn("failed to search DB", "err", err)
		//TODO: add error handling
	}

	for i, b := range books.Items {
		b.CoverPath = strings.TrimPrefix(b.CoverPath, app.bookDir)
		books.Items[i] = b
	}

	w.Header().Set("Content-Type", "application/json")

	js, _ := json.Marshal(books)
	w.Write(js)

}

func (app *booksingApp) downloadBook(w http.ResponseWriter, r *http.Request) {

	hash := r.URL.Query().Get("hash")

	book, err := app.searchDB.GetBook(hash)
	if err != nil {
		slog.Error("could not find book", "err", err, "hash", hash)
		return
	}

	fName := path.Base(book.Path)
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", fName))
	http.ServeFile(w, r, book.Path)
}
