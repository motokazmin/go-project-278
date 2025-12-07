package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"

	"urlcutter/internal/db"
	"urlcutter/internal/links"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	baseURL := os.Getenv("BASE_URL")

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	conn, err := openDB(dbURL)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer conn.Close()

	queries := db.New(conn)
	repo := links.NewRepo(queries)
	service := links.NewService(repo, baseURL)
	handler := links.NewHandler(service)

	router := gin.Default()

	corsOrigin := os.Getenv("FRONTEND_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "http://localhost:5173"
	}
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{corsOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Range", "Authorization"},
		ExposeHeaders:    []string{"Content-Range"},
		AllowCredentials: false,
	}))

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

	pingHandler := func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	}
	router.GET("/ping", pingHandler)
	router.HEAD("/ping", pingHandler)

	handler.Register(router)

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

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
