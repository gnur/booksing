package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gnur/booksing"
	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"
)

func (app *booksingApp) addAPIKey(c *gin.Context) {
	var a booksing.Apikey
	if err := c.ShouldBind(&a); err != nil {
		log.WithField("err", err).Warning("could not get values from post")
		c.JSON(400, gin.H{
			"text": err.Error(),
		})
		return
	}
	log.WithField("keyid", a.ID).Info("adding key")
	a.Created = time.Now().In(app.timezone)
	a.Key = uuid.Must(uuid.NewV4()).String()
	user := c.MustGet("id")
	username := user.(*booksing.User).Username
	a.Username = username

	if err := app.db.SaveAPIKey(&a); err != nil {
		log.WithField("err", err).Warning("could not save api key")
		c.JSON(400, gin.H{
			"text": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"key": a,
	})
}

func (app *booksingApp) deleteAPIKey(c *gin.Context) {
	id := c.Param("uuid")
	if id == "" {
		c.JSON(400, gin.H{
			"text": "invalid uuid provided",
		})
		c.Abort()
		return
	}

	if err := app.db.DeleteAPIKey(id); err != nil {
		c.JSON(500, gin.H{
			"text": err.Error(),
		})
		c.Abort()
		return
	}
	c.JSON(200, gin.H{
		"text": "ok",
	})
}

func (app *booksingApp) getAPIKeys(c *gin.Context) {
	user, _ := c.Get("id")
	username := user.(*booksing.User).Username

	var u struct {
		APIKeys []booksing.Apikey
	}

	apikeys, err := app.db.GetAPIKeysForUser(username)
	if err != nil {
		app.logger.WithField("err", err).Error("could not get apikeys for user")
	}
	u.APIKeys = apikeys

	c.JSON(200, gin.H{
		"user": u,
	})
}
