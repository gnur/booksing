package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type configuration struct {
	AcceptedLanguages []string `default:""`
	BindAddress       string   `default:":7132"`
	MeiliAddress      string   `default:"http://localhost:7700"`
	MeiliSecret       string   `default:""`
	BookDir           string   `default:"./books/"`
	FailDir           string   `default:"./failed"`
	ImportDir         string   `default:"./import"`
	LogLevel          string   `default:"info"`
	MaxSize           int64    `default:"0"`
	Timezone          string   `default:"Europe/Amsterdam"`
}

func main() {
	var cfg configuration
	err := envconfig.Process("booksing", &cfg)
	if err != nil {
		slog.Error("Could not parse full config from environment", "err", err)
		return
	}

	slog.Info("Starting booksing")

	var search searchDB
	search, err = NewMeiliSearch(cfg.MeiliAddress, cfg.MeiliSecret, "booksDev")
	if err != nil {
		slog.Error("could not create meili search", "err", err)
		return
	}

	slog.Info("Started meili integration")

	tz, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		slog.Error("could not load timezone", "err", err)
		return
	}

	slog.Info("Loaded timezone")

	app := booksingApp{
		searchDB:  search,
		bookDir:   cfg.BookDir,
		importDir: cfg.ImportDir,
		timezone:  tz,
		cfg:       cfg,
	}

	if cfg.ImportDir != "" {
		slog.Info("Starting refresh loop", "importDir", cfg.ImportDir)
		go app.refreshLoop()
	}

	port := os.Getenv("PORT")

	mux := http.NewServeMux()
	mux.HandleFunc("/_nuxt/", static)
	mux.HandleFunc("/api/search", app.searchAPI)
	mux.HandleFunc("/book.png", bookPNG)
	mux.HandleFunc("/api/cover", app.getCover)
	mux.HandleFunc("/api/download", app.downloadBook)
	mux.HandleFunc("/api/count", app.count)
	mux.HandleFunc("/", index)

	if port == "" {
		port = cfg.BindAddress
	} else {
		port = fmt.Sprintf(":%s", port)
	}
	slog.Info("booksing will now start listening", "port", port)

	err = http.ListenAndServe(port, mux)
	if err != nil {
		slog.Error("unable to start running", "err", err)
	}
}

func (app *booksingApp) keepBook(b *Book) bool {
	if b == nil {
		return false
	}

	if app.cfg.MaxSize > 0 && b.Size > app.cfg.MaxSize {
		return false
	}

	if len(app.cfg.AcceptedLanguages) > 0 {
		return contains(app.cfg.AcceptedLanguages, b.Language)
	}

	return true
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if strings.EqualFold(s, needle) {
			return true
		}
	}
	return false
}
