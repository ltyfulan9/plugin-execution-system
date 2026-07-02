package handler

import (
	"net/http"
	"strconv"

	"plugin-execution-system/internal/model"
)

func PageQueryFromRequest(r *http.Request) model.PageQuery {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	return model.NormalizePageQuery(model.PageQuery{Page: page, PageSize: pageSize})
}

func PageItems[T any](items []T, q model.PageQuery) model.PageResult {
	q = model.NormalizePageQuery(q)
	total := len(items)
	start := (q.Page - 1) * q.PageSize
	if start > total {
		start = total
	}
	end := start + q.PageSize
	if end > total {
		end = total
	}
	return model.PageResult{Page: q.Page, PageSize: q.PageSize, Total: total, Items: items[start:end]}
}
