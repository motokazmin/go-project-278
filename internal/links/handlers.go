package links

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"urlcutter/internal/db"
)

type Handler struct {
	service LinkService
}

const invalidIDMessage = "invalid id"
const linkNotFoundMessage = "link not found"

type LinkService interface {
	List(ctx context.Context, offset, limit int32) ([]db.Link, int64, error)
	Get(ctx context.Context, id int64) (db.Link, error)
	Create(ctx context.Context, originalURL, shortName string) (db.Link, error)
	Update(ctx context.Context, id int64, originalURL, shortName string) (db.Link, error)
	Delete(ctx context.Context, id int64) error
}

func NewHandler(service LinkService) *Handler {
	return &Handler{service: service}
}

type linkRequest struct {
	OriginalURL string `json:"original_url"`
	ShortName   string `json:"short_name"`
}

type linkResponse struct {
	ID          int64  `json:"id"`
	OriginalURL string `json:"original_url"`
	ShortName   string `json:"short_name"`
	ShortURL    string `json:"short_url"`
}

func (h *Handler) Register(r *gin.Engine) {
	api := r.Group("/api")
	linksGroup := api.Group("/links")

	linksGroup.GET("", h.listLinks)
	linksGroup.POST("", h.createLink)
	linksGroup.GET("/:id", h.getLink)
	linksGroup.PUT("/:id", h.updateLink)
	linksGroup.DELETE("/:id", h.deleteLink)
}

func (h *Handler) listLinks(c *gin.Context) {
	start, end, err := parseRangeParam(c.Query("range"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid range"})
		return
	}

	limit := end - start + 1
	if limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid range"})
		return
	}

	links, total, err := h.service.List(c.Request.Context(), int32(start), int32(limit))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]linkResponse, 0, len(links))
	for _, l := range links {
		resp = append(resp, toResponse(l))
	}

	lastIndex := start + len(resp) - 1
	if len(resp) == 0 {
		lastIndex = start
	}
	c.Header("Content-Range", formatContentRange(start, lastIndex, total))
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) getLink(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": invalidIDMessage})
		return
	}

	link, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": linkNotFoundMessage})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toResponse(link))
}

func (h *Handler) createLink(c *gin.Context) {
	req := new(linkRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	link, err := h.service.Create(c.Request.Context(), req.OriginalURL, req.ShortName)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "original_url is required"})
			return
		case errors.Is(err, ErrConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "short_name already exists"})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, toResponse(link))
}

func (h *Handler) updateLink(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": invalidIDMessage})
		return
	}

	var req linkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	link, err := h.service.Update(c.Request.Context(), id, req.OriginalURL, req.ShortName)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "original_url and short_name are required"})
			return
		case errors.Is(err, ErrConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "short_name already exists"})
			return
		case errors.Is(err, ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": linkNotFoundMessage})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, toResponse(link))
}

func (h *Handler) deleteLink(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": invalidIDMessage})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": linkNotFoundMessage})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func parseID(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}

func parseRangeParam(raw string) (start int, end int, err error) {
	if strings.TrimSpace(raw) == "" {
		// default: first 10
		return 0, 9, nil
	}

	var arr []int
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return 0, 0, fmt.Errorf("parse range: %w", err)
	}
	if len(arr) != 2 {
		return 0, 0, fmt.Errorf("range must contain two numbers")
	}
	start, end = arr[0], arr[1]
	if start < 0 || end < start {
		return 0, 0, fmt.Errorf("invalid range boundaries")
	}
	return start, end, nil
}

func formatContentRange(start, end int, total int64) string {
	return fmt.Sprintf("links %d-%d/%d", start, end, total)
}

func toResponse(l db.Link) linkResponse {
	return linkResponse{
		ID:          l.ID,
		OriginalURL: l.OriginalUrl,
		ShortName:   l.ShortName,
		ShortURL:    l.ShortUrl,
	}
}
