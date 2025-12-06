package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// setupRouter — вспомогательная функция для настройки роутера Gin
func setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	// Маршрут, который мы тестируем
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	return router
}

func TestPingRoute(t *testing.T) {
	router := setupRouter()

	// Создаем симулированный HTTP-запрос
	// Метод: GET, URL: /ping, Тело: nil
	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Ожидался статус 200 OK")

	assert.Equal(t, "pong", w.Body.String(), "Ожидалось тело ответа 'pong'")
}
