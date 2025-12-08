package links

import (
	"bytes"
	"context"
	"database/sql"
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
	apiLinksPath      = "/api/links"
	apiLinkVisitsPath = "/api/link_visits"
	contentJSON       = "application/json"
)

type stubService struct {
	listFn        func(ctx context.Context, offset, limit int32) ([]db.Link, int64, error)
	getFn         func(ctx context.Context, id int64) (db.Link, error)
	getByCodeFn   func(ctx context.Context, shortName string) (db.Link, error)
	createFn      func(ctx context.Context, originalURL, shortName string) (db.Link, error)
	updateFn      func(ctx context.Context, id int64, originalURL, shortName string) (db.Link, error)
	deleteFn      func(ctx context.Context, id int64) error
	createVisitFn func(ctx context.Context, linkID int64, ip, userAgent, referer string, status int) (db.LinkVisit, error)
	listVisitsFn  func(ctx context.Context, offset, limit int32) ([]db.LinkVisit, int64, error)
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
func (s *stubService) GetByShortName(ctx context.Context, shortName string) (db.Link, error) {
	if s.getByCodeFn == nil {
		return db.Link{}, nil
	}
	return s.getByCodeFn(ctx, shortName)
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
func (s *stubService) CreateVisit(ctx context.Context, linkID int64, ip, userAgent, referer string, status int) (db.LinkVisit, error) {
	if s.createVisitFn == nil {
		return db.LinkVisit{}, nil
	}
	return s.createVisitFn(ctx, linkID, ip, userAgent, referer, status)
}
func (s *stubService) ListVisits(ctx context.Context, offset, limit int32) ([]db.LinkVisit, int64, error) {
	if s.listVisitsFn == nil {
		return nil, 0, nil
	}
	return s.listVisitsFn(ctx, offset, limit)
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

	req := httptest.NewRequest(http.MethodGet, apiLinksPath, nil)
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
	req := httptest.NewRequest(http.MethodPost, apiLinksPath, body)
	req.Header.Set(headerContentType, contentJSON)
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
	req.Header.Set(headerContentType, contentJSON)
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
	req := httptest.NewRequest(http.MethodPost, apiLinksPath, body)
	req.Header.Set(headerContentType, contentJSON)
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

func TestRedirectSuccess(t *testing.T) {
	expectedURL := "https://example.com/abc"
	var capturedVisit db.LinkVisit

	svc := &stubService{
		getByCodeFn: func(ctx context.Context, shortName string) (db.Link, error) {
			assert.Equal(t, "abc", shortName)
			return db.Link{ID: 10, OriginalUrl: expectedURL, ShortName: "abc"}, nil
		},
		createVisitFn: func(ctx context.Context, linkID int64, ip, userAgent, referer string, status int) (db.LinkVisit, error) {
			capturedVisit = db.LinkVisit{
				LinkID:    linkID,
				Ip:        ip,
				UserAgent: sql.NullString{String: userAgent, Valid: userAgent != ""},
				Referer:   sql.NullString{String: referer, Valid: referer != ""},
				Status:    int32(status),
			}
			return capturedVisit, nil
		},
	}
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/r/abc", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Referer", "https://ref.example")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, expectedURL, w.Header().Get("Location"))
	assert.Equal(t, int64(10), capturedVisit.LinkID)
	assert.Equal(t, "test-agent", capturedVisit.UserAgent.String)
	assert.Equal(t, "https://ref.example", capturedVisit.Referer.String)
	assert.Equal(t, int32(http.StatusFound), capturedVisit.Status)
}

func TestRedirectNotFound(t *testing.T) {
	svc := &stubService{
		getByCodeFn: func(ctx context.Context, shortName string) (db.Link, error) {
			return db.Link{}, ErrNotFound
		},
		createVisitFn: func(ctx context.Context, linkID int64, ip, userAgent, referer string, status int) (db.LinkVisit, error) {
			t.Fatalf("visit should not be created on not found")
			return db.LinkVisit{}, nil
		},
	}
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/r/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListLinkVisits(t *testing.T) {
	svc := &stubService{
		listVisitsFn: func(ctx context.Context, offset, limit int32) ([]db.LinkVisit, int64, error) {
			assert.Equal(t, int32(0), offset)
			assert.Equal(t, int32(10), limit)
			return []db.LinkVisit{
				{ID: 1, LinkID: 2, Ip: "1.1.1.1", UserAgent: sql.NullString{String: "ua", Valid: true}, Referer: sql.NullString{String: "ref", Valid: true}, Status: 302},
				{ID: 2, LinkID: 3, Ip: "2.2.2.2", Status: 404},
			}, 2, nil
		},
	}
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, apiLinkVisitsPath, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "link_visits 0-1/2", w.Header().Get("Content-Range"))
	assert.Contains(t, w.Body.String(), `"ip":"1.1.1.1"`)
	assert.Contains(t, w.Body.String(), `"status":404`)
}
