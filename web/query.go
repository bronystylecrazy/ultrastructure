package web

import "time"

type PageQuery struct {
	Page  int `query:"page" json:"page"`
	Limit int `query:"limit" json:"limit"`
}

type DateQuery struct {
	FromDate time.Time `query:"from_date" json:"from_date"`
	ToDate   time.Time `query:"to_date" json:"to_date"`
}
