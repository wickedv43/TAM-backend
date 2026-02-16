package common

import (
	"context"
	"errors"
)

type PageOptions struct {
	Page  int    `query:"page_options.page" json:"page"`
	Take  int    `query:"page_options.take" json:"take"`
	Order string `query:"page_options.order" json:"order"`
}

type Pagination[T any] struct {
	Data  T   `json:"data"`
	Count int `json:"count"`
}

func NewPagination[T any](data T, count int) *Pagination[T] {
	return &Pagination[T]{
		Data:  data,
		Count: count,
	}
}

type QueryBuilder[Entity any, AbstractQB any] interface {
	Offset(offset int) AbstractQB
	Limit(limit int) AbstractQB
	Count(ctx context.Context) (int, error)
	All(ctx context.Context) ([]Entity, error)
}

const (
	maxTake     = 100
	defaultPage = 1
	defaultTake = 20
)

func ValidateAndNormalizePageOptions(page, take int, order string) (int, int, string, error) {
	if page <= 0 {
		page = defaultPage
	}
	if take <= 0 {
		take = defaultTake
	}

	if take > maxTake {
		return 0, 0, "", errors.New("maximum 100 items per page allowed")
	}

	if order == "" {
		order = "DESC"
	}
	if order != "ASC" && order != "DESC" {
		return 0, 0, "", errors.New("order must be ASC or DESC")
	}

	return page, take, order, nil
}

func Paginate[Entity any, AbstractQB any](
	ctx context.Context,
	queryBuilder QueryBuilder[Entity, AbstractQB],
	pageOpts PageOptions,
) (*Pagination[[]Entity], error) {
	page := pageOpts.Page
	take := pageOpts.Take

	if page <= 0 {
		page = 1
	}
	if take <= 0 {
		take = 20
	}
	if take > maxTake {
		return nil, errors.New("maximum items per page exceeded")
	}

	count, err := queryBuilder.Count(ctx)
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * take
	queryBuilder.Offset(offset)
	queryBuilder.Limit(take)

	data, err := queryBuilder.All(ctx)
	if err != nil {
		return nil, err
	}

	return NewPagination(data, count), nil
}
