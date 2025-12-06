package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		log.Println("Sentry disabled: SENTRY_DSN is not set")
	} else {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			TracesSampleRate: 1.0,
		}); err != nil {
			log.Fatalf("failed to init Sentry: %v", err)
		}
		router.Use(sentrygin.New(sentrygin.Options{}))
	}

	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	router.GET("/debug-sentry", func(c *gin.Context) {
		hub := sentrygin.GetHubFromContext(c)
		if hub == nil {
			c.JSON(http.StatusOK, gin.H{"status": "sentry disabled"})
			return
		}

		hub.CaptureException(errors.New("debug Sentry error"))
		hub.Flush(2 * time.Second)

		c.JSON(http.StatusInternalServerError, gin.H{"status": "error captured"})
	})

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
