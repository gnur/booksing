package main

import (
	"errors"
	"math"
	"strings"
	"time"

	"firebase.google.com/go/auth"
	"github.com/gin-gonic/gin"
	"github.com/gnur/booksing"
	"github.com/sirupsen/logrus"
)

var timeFormat = time.RFC3339

// Logger is the logrus logger handler
func Logger(log *logrus.Entry) gin.HandlerFunc {

	return func(c *gin.Context) {
		// other handler can change c.Path so:
		path := c.Request.URL.Path
		start := time.Now()
		c.Next()
		stop := time.Since(start)
		latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		clientUserAgent := c.Request.UserAgent()
		referer := c.Request.Referer()
		dataLength := c.Writer.Size()
		if dataLength < 0 {
			dataLength = 0
		}

		entry := log.WithFields(logrus.Fields{
			"statusCode": statusCode,
			"latency":    latency, // time to process
			"clientIP":   clientIP,
			"method":     c.Request.Method,
			"path":       path,
			"referer":    referer,
			"dataLength": dataLength,
			"userAgent":  clientUserAgent,
		})

		if len(c.Errors) > 0 {
			entry.Error(c.Errors.ByType(gin.ErrorTypePrivate).String())
		} else {
			msg := "GIN handled request"
			if statusCode > 499 {
				entry.Error(msg)
			} else if statusCode > 399 {
				entry.Warn(msg)
			} else {
				entry.Info(msg)
			}
		}
	}
}

func (app *booksingApp) APIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apikey := c.GetHeader("x-api-key")
		if apikey == "" {
			c.JSON(403, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		}
		key, err := app.db.GetAPIKey(apikey)
		if err == booksing.ErrNotFound {
			c.JSON(403, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		} else if err != nil {
			c.JSON(500, gin.H{
				"msg": err.Error(),
			})
			c.Abort()
			return
		}
		key.LastUsed = time.Now().In(app.timezone)
		app.db.SaveAPIKey(key)

		c.Set("apikey", key.ID)
		c.Set("id", &booksing.User{
			Username: key.Username,
		})

	}
}

func (app *booksingApp) BearerTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		rawCookie, _ := c.Cookie("Authorization")
		rawHeader := c.GetHeader("Authorization")
		app.logger.WithFields(logrus.Fields{
			"cookie": rawCookie,
			"header": rawHeader,
		}).Debug("incoming request")

		token, err := app.checkCookieAndHeader(rawHeader, rawCookie)
		if err != nil {
			app.logger.WithFields(logrus.Fields{
				"err":    err,
				"cookie": rawCookie,
				"header": rawHeader,
			}).Error("could not validate provided bearer token")
			c.JSON(403, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		}

		user, ok := token.Claims["email"]
		if !ok {
			app.logger.Error("could not extract email from claims")
			c.JSON(403, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		}

		username, ok := user.(string)
		if !ok {
			app.logger.Error("could not cast user to string")
			c.JSON(403, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		}

		u, err := app.db.GetUser(username)
		if err == booksing.ErrNotFound {
			err = app.db.SaveUser(&booksing.User{
				Username:  username,
				IsAdmin:   username == app.adminUser,
				IsAllowed: username == app.adminUser,
				Created:   time.Now(),
				LastSeen:  time.Now(),
			})
			if err != nil {
				app.logger.WithField("err", err).Error("could not save new user")
				c.JSON(500, gin.H{
					"msg": "internal server error",
				})
				c.Abort()
				return
			}
		} else if err == nil {
			u.LastSeen = time.Now()
			err = app.db.SaveUser(&u)
			if err != nil {
				app.logger.Error("could not update user")
				c.JSON(500, gin.H{
					"msg": "internal server error",
				})
				c.Abort()
				return
			}
		} else {
			app.logger.WithField("err", err).Error("could not get user")
			c.JSON(500, gin.H{
				"msg": "internal server error",
			})
			c.Abort()
			return
		}
		if !u.IsAllowed {
			c.JSON(430, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		}

		c.Set("id", &u)

	}
}

func (app *booksingApp) CronMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rawHeader := c.GetHeader("X-Appengine-Cron")
		if rawHeader != "true" {
			app.logger.Warning("cron called without valid header")
			c.JSON(403, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		}

	}
}

func (app *booksingApp) checkCookieAndHeader(h, c string) (*auth.Token, error) {
	t, err := app.checkCookie(c)
	if err == nil {
		return t, nil
	}
	t, err = app.checkHeader(h)
	if err == nil {
		return t, nil
	}
	return nil, errors.New("invalid request")
}

func (app *booksingApp) checkHeader(rawHeader string) (*auth.Token, error) {
	if rawHeader == "" {
		return nil, errors.New("empty header")
	}
	if !strings.HasPrefix(rawHeader, "Bearer ") {
		return nil, errors.New("invalid header")
	}

	jwt := strings.TrimPrefix(rawHeader, "Bearer ")

	return app.validateToken(jwt)
}

func (app *booksingApp) checkCookie(c string) (*auth.Token, error) {

	return app.validateToken(c)
}
