package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/gnur/booksing/firestore"
	_ "github.com/gnur/booksing/firestore"

	//	"github.com/gnur/booksing/mongodb"
	//	_ "github.com/gnur/booksing/mongodb"
	//	"github.com/gnur/booksing/storm"
	//	_ "github.com/gnur/booksing/storm"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	ginlogrus "github.com/toorop/gin-logrus"
)

type configuration struct {
	AllowDeletes  bool
	AllowOrganize bool
	BookDir       string `default:"."`
	ImportDir     string `default:"./import"`
	Database      string `default:"file://booksing.db"`
	LogLevel      string `default:"info"`
	BindAddress   string `default:"localhost:7132"`
	Version       string `default:"unknown"`
	Timezone      string `default:"Europe/Amsterdam"`
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
		//		db, err = mongodb.New(cfg.Database)
		//		if err != nil {
		//			log.WithField("err", err).Fatal("could not create mongodb connection")
		//		}
	} else if strings.HasPrefix(cfg.Database, "firestore://") {
		log.WithField("project", cfg.Database).Debug("using firestore")
		project := strings.TrimPrefix(cfg.Database, "firestore://")
		db, err = firestore.New(project)
		if err != nil {
			log.WithField("err", err).Fatal("could not create firestore client")
		}
	} else if strings.HasPrefix(cfg.Database, "file://") {
		log.WithField("filedbpath", cfg.Database).Debug("using this file")
		//	db, err = storm.New(cfg.Database)
		//	if err != nil {
		//		log.WithField("err", err).Fatal("could not create fileDB")
		//	}
		//	defer db.Close()
	} else {
		log.Fatal("Please set either a mongo host or filedb path")
	}

	app := booksingApp{
		db:            db,
		allowDeletes:  cfg.AllowDeletes,
		allowOrganize: cfg.AllowOrganize,
		bookDir:       cfg.BookDir,
		importDir:     cfg.ImportDir,
		logger:        log.WithField("release", cfg.Version),
	}
	go app.refreshLoop()

	r := gin.New()
	r.Use(ginlogrus.Logger(app.logger), gin.Recovery())

	bfs := BinaryFileSystem("web/dist")
	r.Use(static.Serve("/", bfs))
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.RequestURI()
		if strings.HasPrefix(path, "/auth") || strings.HasPrefix(path, "/api") {
			c.JSON(404, gin.H{
				"msg": "not found",
			})
			return
		}
		b, _ := Asset("web/dist/index.html")
		c.Data(200, "html", b)
	})

	auth := r.Group("/auth")
	auth.Use(gin.Recovery())
	{
		auth.GET("refresh", app.refreshBooks)
		auth.GET("search", app.getBooks)
	}

	http.HandleFunc("user.json", app.getUser())
	http.HandleFunc("downloads.json", app.getDownloads())
	http.HandleFunc("refreshes.json", app.getRefreshes())
	http.HandleFunc("convert/", app.convertBook())
	http.HandleFunc("delete/", app.deleteBook())
	http.HandleFunc("download/", app.downloadBook())

	api := r.Group("/api")
	api.Use(gin.Recovery())
	{
		api.GET("refresh", app.refreshBooks)
		api.GET("exists/:author/:title", app.bookPresent)
	}

	log.Info("booksing is now running")
	port := os.Getenv("PORT")

	if port == "" {
		port = cfg.BindAddress
	} else {
		port = fmt.Sprintf(":%s", port)
	}

	r.Run(port)
}
