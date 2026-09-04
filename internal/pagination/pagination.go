package pagination

import (
	"errors"
	"net/url"
	"strconv"
)

const (
	DefaultPage = 1
	DefaultSize = 10
	MaxSize     = 100
)

var ErrQuery = errors.New("page must be at least 1 and size must be between 1 and 100")

type Query struct {
	Page int
	Size int
}

type Result[T any] struct {
	Items []T
	Query Query
	Total int64
}

func Parse(values url.Values) (Query, error) {
	page, err := number(values.Get("page"), DefaultPage)
	if err != nil || page < 1 {
		return Query{}, ErrQuery
	}
	size, err := number(values.Get("size"), DefaultSize)
	if err != nil || size < 1 || size > MaxSize {
		return Query{}, ErrQuery
	}
	return Query{Page: page, Size: size}, nil
}

func Clamp(query Query, total int64) Query {
	pages := TotalPages(total, query.Size)
	if pages == 0 {
		query.Page = DefaultPage
	} else if query.Page > pages {
		query.Page = pages
	}
	return query
}

func Offset(query Query) int64 {
	return int64(query.Page-1) * int64(query.Size)
}

func TotalPages(total int64, size int) int {
	if total == 0 {
		return 0
	}
	return int((total + int64(size) - 1) / int64(size))
}

func number(value string, fallback int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}
