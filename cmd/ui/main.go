package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gnur/booksing"
	"github.com/gnur/booksing/meili"

	"github.com/kelseyhightower/envconfig"
)

type configuration struct {
	AcceptedLanguages []string `default:""`
	BindAddress       string   `default:":7132"`
	MeiliAddress      string   `default:"http://localhost:7700"`
	BookDir           string   `default:"./books/"`
	FailDir           string   `default:"./failed"`
	ImportDir         string   `default:"./import"`
	LogLevel          string   `default:"info"`
	MaxSize           int64    `default:"0"`
	Timezone          string   `default:"Europe/Amsterdam"`
	UserHeader        string   `default:""`
}

func main() {
	var cfg configuration
	err := envconfig.Process("booksing", &cfg)
	if err != nil {
		slog.Error("Could not parse full config from environment", "err", err)
		return
	}

	var search searchDB
	search, err = meili.New(cfg.MeiliAddress, "", "booksDev")
	if err != nil {
		slog.Error("could not create meili search", "err", err)
		return
	}

	tz, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		slog.Error("could not load timezone", "err", err)
		return
	}

	app := booksingApp{
		searchDB:  search,
		bookDir:   cfg.BookDir,
		importDir: cfg.ImportDir,
		timezone:  tz,
		cfg:       cfg,
	}

	if cfg.ImportDir != "" {
		go app.refreshLoop()
	}

	/*

			static := r.Group("/", func(c *gin.Context) {
				c.Header("Cache-Control", "public, max-age=86400, immutable")
			})

		r.GET("/status", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": app.state,
				"total":  app.searchDB.GetBookCount(),
			})
		})

		auth := r.Group("/")
		auth.Use(app.BearerTokenMiddleware())
		{
			//auth.GET("/", app.search)
			r.GET("/api/search", app.searchAPI)
			r.GET("/detail/:hash", app.detailPage)
			r.GET("/download", app.downloadBook)
			r.GET("/cover", app.cover)

		}

		admin := r.Group("/admin")
		admin.Use(gin.Recovery(), app.BearerTokenMiddleware(), app.mustBeAdmin())
		{
			admin.GET("/users", app.showUsers)
			admin.GET("/downloads", app.showDownloads)
			admin.POST("/delete/:hash", app.deleteBook)
			admin.POST("user/:username", app.updateUser)
			admin.POST("/adduser", app.addUser)
		}

		r.StaticFS("/_nuxt", http.FS(booksing.NuxtElements))
		r.GET("/", func(c *gin.Context) {
			c.Data(200, "text/html", booksing.NuxtIndexHTML)
		})

		// */

	port := os.Getenv("PORT")

	mux := http.NewServeMux()
	mux.HandleFunc("/_nuxt/", static)
	mux.HandleFunc("/api/search", app.searchAPI)
	mux.HandleFunc("/book.png", bookPNG)
	mux.HandleFunc("/cover", app.getCover)
	mux.HandleFunc("/download", app.downloadBook)
	mux.HandleFunc("/", index)

	if port == "" {
		port = cfg.BindAddress
	} else {
		port = fmt.Sprintf(":%s", port)
	}
	slog.Info("booksing is now running", "port", port)

	//err = r.Run(port)
	err = http.ListenAndServe(port, mux)
	if err != nil {
		slog.Error("unable to start running", "err", err)
	}
}

func (app *booksingApp) keepBook(b *booksing.Book) bool {
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
