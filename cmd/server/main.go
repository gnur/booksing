package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gnur/booksing/firestore"
	_ "github.com/gnur/booksing/firestore"
	"github.com/gnur/booksing/mongodb"
	_ "github.com/gnur/booksing/mongodb"
	"github.com/gnur/booksing/storm"
	_ "github.com/gnur/booksing/storm"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

type configuration struct {
	AllowDeletes  bool
	AllowOrganize bool
	BookDir       string `default:"."`
	ImportDir     string `default:""`
	Database      string `default:"file://booksing.db"`
	LogLevel      string `default:"info"`
	BindAddress   string `default:"localhost:7132"`
}

func main() {
	var cfg configuration
	err := envconfig.Process("booksing", &cfg)
	if err != nil {
		log.WithField("err", err).Fatal("Could not parse full config from environment")
	}

	logLevel, err := log.ParseLevel(cfg.LogLevel)
	if err == nil {
		log.SetLevel(logLevel)
	}
	if cfg.ImportDir == "" {
		cfg.ImportDir = path.Join(cfg.BookDir, "import")
	}

	var db database
	if strings.HasPrefix(cfg.Database, "mongo://") {
		log.WithField("mongohost", cfg.Database).Debug("connectiong to mongodb")
		db, err = mongodb.New(cfg.Database)
		if err != nil {
			log.WithField("err", err).Fatal("could not create mongodb connection")
		}
	} else if strings.HasPrefix(cfg.Database, "firestore://") {
		log.WithField("project", cfg.Database).Debug("using firestore")
		project := strings.TrimPrefix(cfg.Database, "firestore://")
		db, err = firestore.New(project)
		if err != nil {
			log.WithField("err", err).Fatal("could not create firestore client")
		}
	} else if strings.HasPrefix(cfg.Database, "file://") {
		log.WithField("filedbpath", cfg.Database).Debug("using this file")
		db, err = storm.New(cfg.Database)
		if err != nil {
			log.WithField("err", err).Fatal("could not create fileDB")
		}
		defer db.Close()
	} else {
		log.Fatal("Please set either a mongo host or filedb path")
	}

	app := booksingApp{
		db:            db,
		allowDeletes:  cfg.AllowDeletes,
		allowOrganize: cfg.AllowOrganize,
		bookDir:       cfg.BookDir,
		importDir:     cfg.ImportDir,
	}
	go app.refreshLoop()

	http.HandleFunc("/api/refresh", app.refreshBooks())
	http.HandleFunc("/api/search", app.getBooks())
	http.HandleFunc("/api/book.json", app.getBook())
	http.HandleFunc("/api/user.json", app.getUser())
	http.HandleFunc("/api/downloads.json", app.getDownloads())
	http.HandleFunc("/api/refreshes.json", app.getRefreshes())
	http.HandleFunc("/api/exists", app.bookPresent())
	http.HandleFunc("/api/convert/", app.convertBook())
	http.HandleFunc("/api/delete/", app.deleteBook())
	http.HandleFunc("/api/download/", app.downloadBook())
	http.Handle("/", http.FileServer(assetFS()))

	log.Info("booksing is now running")
	port := os.Getenv("PORT")

	if port == "" {
		port = cfg.BindAddress
	} else {
		port = fmt.Sprintf(":%s", port)
	}

	log.Fatal(http.ListenAndServe(port, nil))
}
