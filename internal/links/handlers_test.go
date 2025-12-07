package links

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"urlcutter/internal/db"
)

type stubService struct {
	listFn   func(ctx context.Context) ([]db.Link, error)
	getFn    func(ctx context.Context, id int64) (db.Link, error)
	createFn func(ctx context.Context, originalURL, shortName string) (db.Link, error)
	updateFn func(ctx context.Context, id int64, originalURL, shortName string) (db.Link, error)
	deleteFn func(ctx context.Context, id int64) error
}

func (s *stubService) List(ctx context.Context) ([]db.Link, error) {
	if s.listFn == nil {
		return nil, nil
	}
	return s.listFn(ctx)
}
func (s *stubService) Get(ctx context.Context, id int64) (db.Link, error) {
	if s.getFn == nil {
		return db.Link{}, nil
	}
	return s.getFn(ctx, id)
}
func (s *stubService) Create(ctx context.Context, originalURL, shortName string) (db.Link, error) {
	if s.createFn == nil {
		return db.Link{}, nil
	}
	return s.createFn(ctx, originalURL, shortName)
}
func (s *stubService) Update(ctx context.Context, id int64, originalURL, shortName string) (db.Link, error) {
	if s.updateFn == nil {
		return db.Link{}, nil
	}
	return s.updateFn(ctx, id, originalURL, shortName)
}
func (s *stubService) Delete(ctx context.Context, id int64) error {
	if s.deleteFn == nil {
		return nil
	}
	return s.deleteFn(ctx, id)
}

func newTestRouter(svc LinkService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	NewHandler(svc).Register(r)
	return r
}

func TestListLinks(t *testing.T) {
	svc := &stubService{
		listFn: func(ctx context.Context) ([]db.Link, error) {
			return []db.Link{
				{ID: 1, OriginalUrl: "https://example.com", ShortName: "ex", ShortUrl: "https://short/ex"},
			}, nil
		},
	}
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/links", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":1`)
}

func TestCreateLinkConflict(t *testing.T) {
	svc := &stubService{
		createFn: func(ctx context.Context, originalURL, shortName string) (db.Link, error) {
			return db.Link{}, ErrConflict
		},
	}
	r := newTestRouter(svc)

	body := bytes.NewBufferString(`{"original_url":"https://example.com","short_name":"ex"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/links", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestGetLinkNotFound(t *testing.T) {
	svc := &stubService{
		getFn: func(ctx context.Context, id int64) (db.Link, error) {
			return db.Link{}, ErrNotFound
		},
	}
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/links/99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateBadRequest(t *testing.T) {
	svc := &stubService{
		updateFn: func(ctx context.Context, id int64, originalURL, shortName string) (db.Link, error) {
			return db.Link{}, ErrInvalidInput
		},
	}
	r := newTestRouter(svc)

	body := bytes.NewBufferString(`{"original_url":""}`)
	req := httptest.NewRequest(http.MethodPut, "/api/links/1", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteNotFound(t *testing.T) {
	svc := &stubService{
		deleteFn: func(ctx context.Context, id int64) error {
			return ErrNotFound
		},
	}
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/links/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateInvalidPayload(t *testing.T) {
	svc := &stubService{
		createFn: func(ctx context.Context, originalURL, shortName string) (db.Link, error) {
			return db.Link{}, ErrInvalidInput
		},
	}
	r := newTestRouter(svc)

	body := bytes.NewBufferString(`{"original_url":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/links", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUnexpectedError(t *testing.T) {
	svc := &stubService{
		getFn: func(ctx context.Context, id int64) (db.Link, error) {
			return db.Link{}, errors.New("boom")
		},
	}
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/links/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
