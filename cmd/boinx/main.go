package main

import (
	"github.com/jaffee/commandeer"
	log "github.com/sirupsen/logrus"
)

// Configuration bla
type Configuration struct {
	Bucket    string `help:"What bucket is used to store the lambda code zips?"`
	ImportDir string `help:"Directory to load books from"`
}

func newConfig() *Configuration {
	return &Configuration{
		ImportDir: "",
	}
}

var runtimes = []string{
	"go1.x",
}

var cfg Configuration

func main() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "15:04:05.999"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	err := commandeer.Run(newConfig())
	if err != nil {
		log.WithField("err", err).Error("failed")
	}

}

// Run does the actual thingies
func (cfg *Configuration) Run() error {
	errors := false
	if cfg.Bucket == "" {
		log.Error("please provide the bucket name to upload to")
		errors = true
	}
	if errors {
		log.Fatal("Invalid information provided")
	}

	return nil
}
