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

const (
	headerContentType = "Content-Type"
)

type stubService struct {
	listFn   func(ctx context.Context, offset, limit int32) ([]db.Link, int64, error)
	getFn    func(ctx context.Context, id int64) (db.Link, error)
	createFn func(ctx context.Context, originalURL, shortName string) (db.Link, error)
	updateFn func(ctx context.Context, id int64, originalURL, shortName string) (db.Link, error)
	deleteFn func(ctx context.Context, id int64) error
}

func (s *stubService) List(ctx context.Context, offset, limit int32) ([]db.Link, int64, error) {
	if s.listFn == nil {
		return nil, 0, nil
	}
	return s.listFn(ctx, offset, limit)
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
		listFn: func(ctx context.Context, offset, limit int32) ([]db.Link, int64, error) {
			return []db.Link{
				{ID: 1, OriginalUrl: "https://example.com", ShortName: "ex", ShortUrl: "https://short/ex"},
			}, 1, nil
		},
	}
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/links", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":1`)
	assert.Equal(t, "links 0-0/1", w.Header().Get("Content-Range"))
}

func TestListLinksWithRange(t *testing.T) {
	svc := &stubService{
		listFn: func(ctx context.Context, offset, limit int32) ([]db.Link, int64, error) {
			assert.Equal(t, int32(5), offset)
			assert.Equal(t, int32(5), limit)
			return []db.Link{
				{ID: 6, OriginalUrl: "https://example.com/6", ShortName: "s6", ShortUrl: "http://localhost/r/s6"},
				{ID: 7, OriginalUrl: "https://example.com/7", ShortName: "s7", ShortUrl: "http://localhost/r/s7"},
			}, 11, nil
		},
	}
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/links?range=[5,9]", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "links 5-6/11", w.Header().Get("Content-Range"))
}

func TestListLinksInvalidRange(t *testing.T) {
	svc := &stubService{}
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/links?range=bad", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
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
	req.Header.Set(headerContentType, "application/json")
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
	req.Header.Set(headerContentType, "application/json")
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
	req.Header.Set(headerContentType, "application/json")
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
