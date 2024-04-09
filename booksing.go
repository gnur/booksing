package main

import (
	"context"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	zglob "github.com/mattn/go-zglob"
	"golang.org/x/sync/semaphore"
)

const (
	stateUnlocked uint32 = iota
	stateLocked
)

var (
	locker = stateUnlocked
)

func (app *booksingApp) refreshLoop() {
	app.refresh()
	for {
		select {
		case <-time.After(time.Minute):
		case <-app.refreshChan:
			app.refresh()
		}
	}
}

func (app *booksingApp) refresh() {
	if !atomic.CompareAndSwapUint32(&locker, stateUnlocked, stateLocked) {
		slog.Warn("not refreshing because it is already running")
		return
	}
	slog.Info("Scanning import dir")
	defer atomic.StoreUint32(&locker, stateUnlocked)
	defer func() {
		app.state = "idle"
	}()

	app.state = "indexing"
	matches, err := zglob.Glob(filepath.Join(app.importDir, "/**/*.epub"))
	if err != nil {
		slog.Error("glob of all books failed", "err", err)
		return
	}

	if len(matches) == 0 {
		slog.Info("no new books found")
		return
	}
	var books []Book
	counter := 0

	slog.Info("located books on filesystem, processing per batchsize", "total", len(matches), "bookdir", app.importDir)

	ctx := context.TODO()
	toProcess := len(matches)
	bookQ := make(chan *Book)
	sem := semaphore.NewWeighted(int64(runtime.GOMAXPROCS(0)))

	for _, filename := range matches {
		slog.Debug("parsing book", "f", filename)

		go func(f string) {
			if err := sem.Acquire(ctx, 1); err != nil {
				slog.Error("failed to acquire semaphore", "err", err)
			}
			defer sem.Release(1)

			book, err := NewBookFromFile(f, app.bookDir)
			if err != nil {
				slog.Error("failed to parse book", "err", err, "file", f)
				app.moveBookToFailed(f)
			}

			bookQ <- book
		}(filename)

	}
	processed := 0
	for book := range bookQ {
		processed++
		if book == nil {
			if processed == toProcess {
				close(bookQ)
			}
			continue
		}
		if !app.keepBook(book) {
			app.moveBookToFailed(book.Path)
			if processed == toProcess {
				close(bookQ)
			}
			continue
		}
		books = append(books, *book)
		counter++
		if len(books) == 50 || processed == toProcess {
			err = app.searchDB.AddBooks(books)
			if err != nil {
				slog.Error("bulk insert into meili failed", "err", err)
			}
			books = []Book{}
		}
		slog.Debug("processed book", "counter", counter, "total", toProcess)
		if processed == toProcess {
			close(bookQ)
		}

	}
	if len(books) > 0 {
		slog.Error("This should absolutely not happen")
		err = app.searchDB.AddBooks(books)
		if err != nil {
			slog.Info("bulk insert into meili failed", "err", err)
		}
	}

	slog.Info("Done with refresh")

}

func (app *booksingApp) moveBookToFailed(bookpath string) {
	err := os.MkdirAll(app.cfg.FailDir, 0755)
	if err != nil {
		slog.Error("unable to create fail dir", "err", err)
		return
	}
	globPath := strings.Replace(bookpath, ".epub", ".*", 1)
	files, err := zglob.Glob(globPath)
	if err != nil {
		return
	}

	for _, f := range files {
		filename := path.Base(f)
		newBookPath := path.Join(app.cfg.FailDir, filename)
		err = os.Rename(bookpath, newBookPath)
		if err != nil {
			slog.Error("unable to move book to faildir", "err", err, "faildir", app.cfg.FailDir, "bookpath", bookpath)
		}
	}
}
