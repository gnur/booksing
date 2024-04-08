package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

func static(w http.ResponseWriter, r *http.Request) {
	fsPublic, _ := fs.Sub(NuxtElements, "web/.output/public")
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
	_, err := w.Write(NuxtIndexHTML)
	if err != nil {
		slog.Warn("failed to write index", "err", err)
	}
}

func bookPNG(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write(BookPNG)
	if err != nil {
		slog.Warn("failed to write index", "err", err)
	}
}

type countResult struct {
	Total int `json:"total"`
}

func (app *booksingApp) count(w http.ResponseWriter, r *http.Request) {
	count := app.searchDB.GetBookCount()
	js, err := json.Marshal(countResult{Total: count})
	if err != nil {
		slog.Warn("failed to marshal count", "err", err)
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		slog.Warn("failed to write index", "err", err)
	}
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
	limit = 9
	q := r.URL.Query().Get("q")
	off := r.URL.Query().Get("o")
	if off != "" {
		offset, err = strconv.ParseInt(off, 10, 64)
		if err != nil {
			offset = 0
		}
	}
	if lim := r.URL.Query().Get("l"); lim != "" {
		limit, err = strconv.ParseInt(lim, 10, 64)
		if err != nil {
			limit = 20
		}
	}

	var books *SearchResult

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
	_, err = w.Write(js)
	if err != nil {
		slog.Warn("failed to write search result", "err", err)
	}

}

func (app *booksingApp) downloadBook(w http.ResponseWriter, r *http.Request) {

	hash := r.URL.Query().Get("hash")

	book, err := app.searchDB.GetBook(hash)
	if err != nil {
		slog.Error("could not find book", "err", err, "hash", hash)
		return
	}

	if app.webHookEnabled {
		//do this async so the user does not have to wait for the webhook to finish
		go app.fireWebHook(webHookData{
			IPs:  getIPFromRequest(r),
			User: getUserFromRequest(r),
			Hash: hash,
		})
	}

	fName := path.Base(book.Path)
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", fName))
	http.ServeFile(w, r, book.Path)
}

const maxUploadSize = 20 * 1024 * 1024 // 2 mb

func (app *booksingApp) addBook(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		fmt.Printf("Could not parse multipart form: %v\n", err)
		renderError(w, "CANT_PARSE_FORM", http.StatusInternalServerError)
		return
	}

	// parse and validate file and post parameters
	file, fileHeader, err := r.FormFile("uploadFile")
	if err != nil {
		renderError(w, "INVALID_FILE", http.StatusBadRequest)
		return
	}
	defer file.Close()
	// Get and print out file size
	fileSize := fileHeader.Size
	fmt.Printf("File size (bytes): %v\n", fileSize)
	// validate file size
	if fileSize > maxUploadSize {
		renderError(w, "FILE_TOO_BIG", http.StatusBadRequest)
		return
	}
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		renderError(w, "INVALID_FILE", http.StatusBadRequest)
		return
	}

	// check file type, detectcontenttype only needs the first 512 bytes
	detectedFileType := http.DetectContentType(fileBytes)
	slog.Info("File type detected as ", "filetype", detectedFileType)
	switch detectedFileType {
	case "application/zip":
		break
	default:
		renderError(w, "INVALID_FILE_TYPE", http.StatusBadRequest)
		return
	}
	fileName := randToken(12)
	fileEndings, err := mime.ExtensionsByType(detectedFileType)
	if err != nil {
		renderError(w, "CANT_READ_FILE_TYPE", http.StatusInternalServerError)
		return
	}
	if fileEndings[0] != ".zip" {
		return
	}

	newFileName := fileName + ".epub"
	newPath := filepath.Join(app.importDir, newFileName)
	fmt.Printf("FileType: %s, File: %s\n", detectedFileType, newPath)
	// write file
	newFile, err := os.Create(newPath)
	if err != nil {
		renderError(w, "CANT_WRITE_FILE", http.StatusInternalServerError)
		return
	}
	defer newFile.Close() // idempotent, okay to call twice
	if _, err := newFile.Write(fileBytes); err != nil || newFile.Close() != nil {
		renderError(w, "CANT_WRITE_FILE", http.StatusInternalServerError)
		return
	}
	app.refreshChan <- true
	w.Write([]byte("SUCCESS"))
}
func renderError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	w.Write([]byte(message))
}

func randToken(len int) string {
	b := make([]byte, len)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
