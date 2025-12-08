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

func (r SQLRepo) ListLinksRange(ctx context.Context, offset int32, limit int32) ([]db.Link, error) {
	return r.Queries.ListLinksRange(ctx, db.ListLinksRangeParams{
		Offset: offset,
		Limit:  limit,
	})
}

func (r SQLRepo) Link(ctx context.Context, id int64) (db.Link, error) {
	return r.Queries.Link(ctx, id)
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

func (r SQLRepo) CountLinks(ctx context.Context) (int64, error) {
	return r.Queries.CountLinks(ctx)
}
