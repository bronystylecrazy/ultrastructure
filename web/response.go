package web

type PagedMeta struct {
	Page       int  `json:"page,omitempty"`
	Limit      int  `json:"limit,omitempty"`
	TotalItems int  `json:"total_items,omitempty"`
	TotalPages int  `json:"total_pages,omitempty"`
	HasNext    bool `json:"has_next,omitempty"`
	HasPrev    bool `json:"has_prev,omitempty"`
}

type Response struct {
	Data any `json:"data,omitempty"`
	Meta any `json:"meta,omitempty"`
}

type PagedResponse struct {
	Data any       `json:"data,omitempty"`
	Meta PagedMeta `json:"meta"`
}
