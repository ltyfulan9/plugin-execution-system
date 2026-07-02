package model

type PageQuery struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type PageResult struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
	Items    any `json:"items"`
}

func NormalizePageQuery(q PageQuery) PageQuery {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
	return q
}
