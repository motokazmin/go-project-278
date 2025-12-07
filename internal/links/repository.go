package links

import (
	"context"

	"urlcutter/internal/db"
)

type SQLRepo struct {
	*db.Queries
}

func NewRepo(q *db.Queries) SQLRepo {
	return SQLRepo{Queries: q}
}

func (r SQLRepo) ListLinks(ctx context.Context) ([]db.Link, error) {
	return r.Queries.ListLinks(ctx)
}

func (r SQLRepo) GetLink(ctx context.Context, id int64) (db.Link, error) {
	return r.Queries.GetLink(ctx, id)
}

func (r SQLRepo) CreateLink(ctx context.Context, arg db.CreateLinkParams) (db.Link, error) {
	return r.Queries.CreateLink(ctx, arg)
}

func (r SQLRepo) UpdateLink(ctx context.Context, arg db.UpdateLinkParams) (db.Link, error) {
	return r.Queries.UpdateLink(ctx, arg)
}

func (r SQLRepo) DeleteLink(ctx context.Context, id int64) (int64, error) {
	return r.Queries.DeleteLink(ctx, id)
}
