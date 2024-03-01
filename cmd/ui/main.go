package main

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gnur/booksing"
	"github.com/gnur/booksing/meili"
	"github.com/gnur/booksing/sqlite"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFiles embed.FS

// V is the holder struct for all possible template values
type V struct {
	Results    int64
	Error      error
	Books      []booksing.Book
	Book       *booksing.Book
	ExtraPaths []string
	Users      []booksing.User
	Downloads  []booksing.Download
	Q          string
	TimeTaken  int
	IsAdmin    bool
	Username   string
	TotalBooks int
	Limit      int64
	Offset     int64
	Indexing   bool
}

type configuration struct {
	AcceptedLanguages []string `default:""`
	AdminUser         string   `default:"unknown"`
	AllowAllusers     bool     `default:"true"`
	BindAddress       string   `default:":7132"`
	MeiliAddress      string   `default:"http://localhost:7700"`
	BookDir           string   `default:"./books/"`
	DatabaseDir       string   `default:"./db/"`
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
		log.WithField("err", err).Fatal("Could not parse full config from environment")
	}

	logLevel, err := log.ParseLevel(cfg.LogLevel)
	if err == nil {
		log.SetLevel(logLevel)
	}

	var db database
	log.WithField("dbpath", cfg.DatabaseDir).Debug("using this file")
	db, err = sqlite.New(cfg.DatabaseDir)

	if err != nil {
		log.WithField("err", err).Fatal("could not create fileDB")
	}
	defer db.Close()

	var search searchDB
	search, err = meili.New(cfg.MeiliAddress, "", "booksDev")
	if err != nil {
		log.WithField("err", err).Fatal("could not create meili search")
	}

	tz, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.WithField("err", err).Fatal("could not load timezone")
	}

	tpl := template.New("")
	tpl.Funcs(templateFunctions)
	tpl, err = tpl.ParseFS(templateFiles, "templates/*.html")
	if err != nil {
		log.WithField("err", err).Fatal("could not read templates")
		return
	}

	app := booksingApp{
		db:        db,
		searchDB:  search,
		bookDir:   cfg.BookDir,
		importDir: cfg.ImportDir,
		timezone:  tz,
		adminUser: cfg.AdminUser,
		logger:    log.WithField("app", "booksing"),
		cfg:       cfg,
	}

	if cfg.ImportDir != "" {
		go app.refreshLoop()
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(Logger(app.logger), gin.Recovery())
	r.SetHTMLTemplate(tpl)

	static := r.Group("/", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=86400, immutable")
	})

	static.StaticFS("/static", http.FS(staticFiles))

	r.GET("/kill", func(c *gin.Context) {
		app.logger.Fatal("Killing so I get restarted anew")
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
		auth.GET("/", app.search)
		auth.GET("/api/search", app.searchAPI)
		auth.GET("/detail/:hash", app.detailPage)
		auth.GET("/download", app.downloadBook)
		auth.GET("/cover", app.cover)

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

	port := os.Getenv("PORT")

	if port == "" {
		port = cfg.BindAddress
	} else {
		port = fmt.Sprintf(":%s", port)
	}
	log.WithField("port", port).Info("booksing is now running")

	err = r.Run(port)
	if err != nil {
		log.WithField("err", err).Fatal("unable to start running")
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
