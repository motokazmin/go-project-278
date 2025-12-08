package links

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgconn"

	"urlcutter/internal/db"
)

var (
	ErrNotFound      = errors.New("link not found")
	ErrConflict      = errors.New("short_name already exists")
	ErrInvalidInput  = errors.New("invalid input")
	defaultBaseURL   = "http://localhost:8080"
	alphabet         = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	generatedLength  = 8
	maxGenerateTries = 5
)

type Repository interface {
	ListLinksRange(ctx context.Context, offset int32, limit int32) ([]db.Link, error)
	GetLink(ctx context.Context, id int64) (db.Link, error)
	LinkByShortName(ctx context.Context, shortName string) (db.Link, error)
	CreateLink(ctx context.Context, arg db.CreateLinkParams) (db.Link, error)
	UpdateLink(ctx context.Context, arg db.UpdateLinkParams) (db.Link, error)
	DeleteLink(ctx context.Context, id int64) (int64, error)
	CountLinks(ctx context.Context) (int64, error)
	ListLinkVisitsRange(ctx context.Context, offset int32, limit int32) ([]db.LinkVisit, error)
	CountLinkVisits(ctx context.Context) (int64, error)
	CreateLinkVisit(ctx context.Context, arg db.CreateLinkVisitParams) (db.LinkVisit, error)
}

type Service struct {
	repo    Repository
	baseURL string
}

func NewService(repo Repository, baseURL string) *Service {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &Service{repo: repo, baseURL: baseURL}
}

func (s *Service) List(ctx context.Context, offset, limit int32) ([]db.Link, int64, error) {
	items, err := s.repo.ListLinksRange(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountLinks(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) Get(ctx context.Context, id int64) (db.Link, error) {
	link, err := s.repo.GetLink(ctx, id)
	if err != nil {
		return db.Link{}, mapDBError(err)
	}
	return link, nil
}

func (s *Service) GetByShortName(ctx context.Context, shortName string) (db.Link, error) {
	link, err := s.repo.LinkByShortName(ctx, shortName)
	if err != nil {
		return db.Link{}, mapDBError(err)
	}
	return link, nil
}

func (s *Service) CreateVisit(ctx context.Context, linkID int64, ip, userAgent, referer string, status int) (db.LinkVisit, error) {
	visit, err := s.repo.CreateLinkVisit(ctx, db.CreateLinkVisitParams{
		LinkID:    linkID,
		Ip:        ip,
		UserAgent: sql.NullString{String: userAgent, Valid: userAgent != ""},
		Referer:   sql.NullString{String: referer, Valid: referer != ""},
		Status:    int32(status),
	})
	if err != nil {
		return db.LinkVisit{}, mapDBError(err)
	}
	return visit, nil
}

func (s *Service) ListVisits(ctx context.Context, offset, limit int32) ([]db.LinkVisit, int64, error) {
	items, err := s.repo.ListLinkVisitsRange(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountLinkVisits(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) Create(ctx context.Context, originalURL, shortName string) (db.Link, error) {
	if strings.TrimSpace(originalURL) == "" {
		return db.Link{}, ErrInvalidInput
	}

	if strings.TrimSpace(shortName) != "" {
		return s.createOnce(ctx, originalURL, shortName)
	}

	// short name not provided; generate with retries on conflict
	for i := 0; i < maxGenerateTries; i++ {
		name, err := randomString(generatedLength)
		if err != nil {
			return db.Link{}, fmt.Errorf("generate short name: %w", err)
		}
		link, err := s.createOnce(ctx, originalURL, name)
		if err == nil {
			return link, nil
		}
		if errors.Is(err, ErrConflict) {
			continue
		}
		return db.Link{}, err
	}
	return db.Link{}, fmt.Errorf("failed to generate unique short name after %d attempts", maxGenerateTries)
}

func (s *Service) Update(ctx context.Context, id int64, originalURL, shortName string) (db.Link, error) {
	if strings.TrimSpace(originalURL) == "" || strings.TrimSpace(shortName) == "" {
		return db.Link{}, ErrInvalidInput
	}

	shortURL := s.composeShortURL(shortName)
	link, err := s.repo.UpdateLink(ctx, db.UpdateLinkParams{
		ID:          id,
		OriginalUrl: originalURL,
		ShortName:   shortName,
		ShortUrl:    shortURL,
	})
	if err != nil {
		return db.Link{}, mapDBError(err)
	}
	return link, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	_, err := s.repo.DeleteLink(ctx, id)
	return mapDBError(err)
}

func (s *Service) createOnce(ctx context.Context, originalURL, shortName string) (db.Link, error) {
	shortURL := s.composeShortURL(shortName)
	link, err := s.repo.CreateLink(ctx, db.CreateLinkParams{
		OriginalUrl: originalURL,
		ShortName:   shortName,
		ShortUrl:    shortURL,
	})
	if err != nil {
		return db.Link{}, mapDBError(err)
	}
	return link, nil
}

func (s *Service) composeShortURL(shortName string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, shortName)
}

func randomString(length int) (string, error) {
	b := make([]rune, length)
	for i := range b {
		n, err := randInt(len(alphabet))
		if err != nil {
			return "", fmt.Errorf("generate random index: %w", err)
		}
		b[i] = alphabet[n]
	}
	return string(b), nil
}

func randInt(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("max must be positive")
	}

	var b [1]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, fmt.Errorf("read random byte: %w", err)
	}

	return int(b[0]) % max, nil
}

func mapDBError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return ErrConflict
		}
	}
	return err
}
