package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	firebase "firebase.google.com/go"
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
	Mode          string `default:"dev"`
	FQDN          string `default:"localhost:8080"`
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

	// setup firebase for verification
	fb, err := firebase.NewApp(context.Background(), &firebase.Config{
		ProjectID: "",
	})
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	client, err := fb.Auth(context.Background())
	if err != nil {
		log.Fatalf("error initializing firebase auth client: %v\n", err)
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
		db, err = firestore.New(project, cfg.Mode)
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

	tz, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.WithField("err", err).Fatal("could not load timezone")
	}

	app := booksingApp{
		db:            db,
		allowDeletes:  cfg.AllowDeletes,
		allowOrganize: cfg.AllowOrganize,
		bookDir:       cfg.BookDir,
		importDir:     cfg.ImportDir,
		timezone:      tz,
		authClient:    client,
		FQDN:          cfg.FQDN,
		logger:        log.WithField("release", cfg.Version),
	}
	go app.refreshLoop()

	r := gin.New()
	r.Use(Logger(app.logger), gin.Recovery())

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

	r.POST("/checkToken", app.checkToken)

	auth := r.Group("/auth")
	auth.Use(gin.Recovery(), app.BearerTokenMiddleware())
	{
		auth.GET("search", app.getBooks)
		auth.GET("user.json", app.getUser)

		auth.GET("/apikey", app.getAPIKeys)
		auth.POST("/apikey", app.addAPIKey)
		auth.DELETE("/apikey/:uuid", app.deleteAPIKey)

		auth.POST("convert", app.convertBook)
		auth.GET("download", app.downloadBook)

	}

	admin := r.Group("/admin")
	admin.Use(gin.Recovery())
	{
		admin.GET("downloads.json", app.getDownloads)
		admin.GET("refreshes.json", app.getRefreshes)
		admin.POST("refresh", app.refreshBooks)
		admin.POST("delete", app.deleteBook)
	}

	api := r.Group("/api")
	api.Use(gin.Recovery())
	{
		api.GET("exists/:author/:title", app.bookPresent)
		api.PUT("book", app.addBook)
		api.PUT("books", app.addBooks)
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
