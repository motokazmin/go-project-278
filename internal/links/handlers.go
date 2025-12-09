package links

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"urlcutter/internal/db"
)

type Handler struct {
	service LinkService
}

const invalidIDMessage = "invalid id"
const linkNotFoundMessage = "link not found"
const invalidRangeMessage = "invalid range"

type LinkService interface {
	List(ctx context.Context, offset, limit int32) ([]db.Link, int64, error)
	Get(ctx context.Context, id int64) (db.Link, error)
	GetByShortName(ctx context.Context, shortName string) (db.Link, error)
	Create(ctx context.Context, originalURL, shortName string) (db.Link, error)
	Update(ctx context.Context, id int64, originalURL, shortName string) (db.Link, error)
	Delete(ctx context.Context, id int64) error
	CreateVisit(ctx context.Context, linkID int64, ip, userAgent, referer string, status int) (db.LinkVisit, error)
	ListVisits(ctx context.Context, offset, limit int32) ([]db.LinkVisit, int64, error)
}

func NewHandler(service LinkService) *Handler {
	return &Handler{service: service}
}

type linkRequest struct {
	OriginalURL string `json:"original_url" binding:"required,url"`
	ShortName   string `json:"short_name" binding:"omitempty,min=3,max=32"`
}

type linkResponse struct {
	ID          int64  `json:"id"`
	OriginalURL string `json:"original_url"`
	ShortName   string `json:"short_name"`
	ShortURL    string `json:"short_url"`
}

type linkVisitResponse struct {
	ID        int64     `json:"id"`
	LinkID    int64     `json:"link_id"`
	CreatedAt time.Time `json:"created_at"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Referer   string    `json:"referer"`
	Status    int32     `json:"status"`
}

// handleBindError handles JSON binding errors and returns appropriate response
func handleBindError(c *gin.Context, err error) {
	// Check if it's a validation error
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		handleValidationError(c, validationErrs)
		return
	}

	// Check if it's a JSON syntax error or EOF
	var syntaxErr *json.SyntaxError
	var unmarshalErr *json.UnmarshalTypeError
	if errors.As(err, &syntaxErr) || errors.As(err, &unmarshalErr) || errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Default to bad request for other binding errors
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
}

// handleValidationError converts validator errors to unified format
func handleValidationError(c *gin.Context, errs validator.ValidationErrors) {
	fieldErrors := make(map[string]string)

	for _, err := range errs {
		fieldName := strings.ToLower(err.Field())
		// Convert field name from struct field to JSON field
		if fieldName == "originalurl" {
			fieldName = "original_url"
		} else if fieldName == "shortname" {
			fieldName = "short_name"
		}

		// Create user-friendly error message
		var message string
		switch err.Tag() {
		case "required":
			message = fmt.Sprintf("Key: 'linkRequest.%s' Error:Field validation for '%s' failed on the 'required' tag", fieldName, fieldName)
		case "url":
			message = fmt.Sprintf("Key: 'linkRequest.%s' Error:Field validation for '%s' failed on the 'url' tag", fieldName, fieldName)
		case "min":
			message = fmt.Sprintf("Key: 'linkRequest.%s' Error:Field validation for '%s' failed on the 'min' tag", fieldName, fieldName)
		case "max":
			message = fmt.Sprintf("Key: 'linkRequest.%s' Error:Field validation for '%s' failed on the 'max' tag", fieldName, fieldName)
		default:
			message = fmt.Sprintf("Key: 'linkRequest.%s' Error:Field validation for '%s' failed on the '%s' tag", fieldName, fieldName, err.Tag())
		}

		fieldErrors[fieldName] = message
	}

	c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": fieldErrors})
}

// handleServiceError converts service errors to appropriate HTTP responses
func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": linkNotFoundMessage})
	case errors.Is(err, ErrConflict):
		// Convert conflict to validation error format
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"errors": map[string]string{
				"short_name": "short name already in use",
			},
		})
	case errors.Is(err, ErrInvalidInput):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid input"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/r/:code", h.redirect)

	api := r.Group("/api")
	linksGroup := api.Group("/links")

	linksGroup.GET("", h.listLinks)
	linksGroup.POST("", h.createLink)
	linksGroup.GET("/:id", h.link)
	linksGroup.PUT("/:id", h.updateLink)
	linksGroup.DELETE("/:id", h.deleteLink)

	api.GET("/link_visits", h.listLinkVisits)
}

func (h *Handler) listLinks(c *gin.Context) {
	start, end, err := parseRangeParam(c.Query("range"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": invalidRangeMessage})
		return
	}

	limit := end - start + 1
	if limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": invalidRangeMessage})
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

func (h *Handler) link(c *gin.Context) {
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
		handleBindError(c, err)
		return
	}

	link, err := h.service.Create(c.Request.Context(), req.OriginalURL, req.ShortName)
	if err != nil {
		handleServiceError(c, err)
		return
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
		handleBindError(c, err)
		return
	}

	link, err := h.service.Update(c.Request.Context(), id, req.OriginalURL, req.ShortName)
	if err != nil {
		handleServiceError(c, err)
		return
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

func (h *Handler) redirect(c *gin.Context) {
	code := c.Param("code")

	link, err := h.service.GetByShortName(c.Request.Context(), code)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": linkNotFoundMessage})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	status := http.StatusFound
	_, err = h.service.CreateVisit(c.Request.Context(), link.ID, clientIP(c), c.GetHeader("User-Agent"), c.GetHeader("Referer"), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Redirect(status, link.OriginalUrl)
}

func (h *Handler) listLinkVisits(c *gin.Context) {
	start, end, err := parseRangeParam(c.Query("range"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": invalidRangeMessage})
		return
	}

	limit := end - start + 1
	if limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": invalidRangeMessage})
		return
	}

	visits, total, err := h.service.ListVisits(c.Request.Context(), int32(start), int32(limit))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]linkVisitResponse, 0, len(visits))
	for _, v := range visits {
		resp = append(resp, toVisitResponse(v))
	}

	lastIndex := start + len(resp) - 1
	if len(resp) == 0 {
		lastIndex = start
	}
	c.Header("Content-Range", formatVisitsContentRange(start, lastIndex, total))
	c.JSON(http.StatusOK, resp)
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

func toVisitResponse(v db.LinkVisit) linkVisitResponse {
	return linkVisitResponse{
		ID:        v.ID,
		LinkID:    v.LinkID,
		CreatedAt: v.CreatedAt,
		IP:        v.Ip,
		UserAgent: v.UserAgent.String,
		Referer:   v.Referer.String,
		Status:    v.Status,
	}
}

func formatVisitsContentRange(start, end int, total int64) string {
	return fmt.Sprintf("link_visits %d-%d/%d", start, end, total)
}

func clientIP(c *gin.Context) string {
	if ip := c.ClientIP(); ip != "" {
		return ip
	}
	return ""
}
