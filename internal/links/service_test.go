package links

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jackc/pgconn"
	"github.com/stretchr/testify/assert"

	"urlcutter/internal/db"
)

type memRepo struct {
	items map[int64]db.Link
	next  int64
}

func newMemRepo() *memRepo {
	return &memRepo{
		items: make(map[int64]db.Link),
		next:  1,
	}
}

func (m *memRepo) ListLinks(ctx context.Context) ([]db.Link, error) {
	out := make([]db.Link, 0, len(m.items))
	for _, v := range m.items {
		out = append(out, v)
	}
	return out, nil
}

func (m *memRepo) GetLink(ctx context.Context, id int64) (db.Link, error) {
	v, ok := m.items[id]
	if !ok {
		return db.Link{}, sql.ErrNoRows
	}
	return v, nil
}

func (m *memRepo) CreateLink(ctx context.Context, arg db.CreateLinkParams) (db.Link, error) {
	for _, v := range m.items {
		if v.ShortName == arg.ShortName {
			return db.Link{}, &pgconn.PgError{Code: "23505"}
		}
	}
	id := m.next
	m.next++
	link := db.Link{
		ID:          id,
		OriginalUrl: arg.OriginalUrl,
		ShortName:   arg.ShortName,
		ShortUrl:    arg.ShortUrl,
	}
	m.items[id] = link
	return link, nil
}

func (m *memRepo) UpdateLink(ctx context.Context, arg db.UpdateLinkParams) (db.Link, error) {
	_, ok := m.items[arg.ID]
	if !ok {
		return db.Link{}, sql.ErrNoRows
	}
	for id, v := range m.items {
		if id != arg.ID && v.ShortName == arg.ShortName {
			return db.Link{}, &pgconn.PgError{Code: "23505"}
		}
	}
	link := db.Link{
		ID:          arg.ID,
		OriginalUrl: arg.OriginalUrl,
		ShortName:   arg.ShortName,
		ShortUrl:    arg.ShortUrl,
	}
	m.items[arg.ID] = link
	return link, nil
}

func (m *memRepo) DeleteLink(ctx context.Context, id int64) (int64, error) {
	if _, ok := m.items[id]; !ok {
		return 0, sql.ErrNoRows
	}
	delete(m.items, id)
	return id, nil
}

func TestCreateGeneratesShortName(t *testing.T) {
	repo := newMemRepo()
	svc := NewService(repo, "https://short.test")

	link, err := svc.Create(context.Background(), "https://example.com", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, link.ShortName)
	assert.Contains(t, link.ShortUrl, link.ShortName)
}

func TestCreateConflict(t *testing.T) {
	repo := newMemRepo()
	svc := NewService(repo, "https://short.test")

	_, err := svc.Create(context.Background(), "https://example.com/1", "same")
	assert.NoError(t, err)

	_, err = svc.Create(context.Background(), "https://example.com/2", "same")
	assert.ErrorIs(t, err, ErrConflict)
}

func TestUpdateNotFound(t *testing.T) {
	repo := newMemRepo()
	svc := NewService(repo, "https://short.test")

	_, err := svc.Update(context.Background(), 42, "https://example.com", "name")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestServiceDeleteNotFound(t *testing.T) {
	repo := newMemRepo()
	svc := NewService(repo, "https://short.test")

	err := svc.Delete(context.Background(), 123)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetNotFound(t *testing.T) {
	repo := newMemRepo()
	svc := NewService(repo, "https://short.test")

	_, err := svc.Get(context.Background(), 99)
	assert.ErrorIs(t, err, ErrNotFound)
}
