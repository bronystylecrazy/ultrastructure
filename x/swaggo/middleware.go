package swaggo

import (
	"github.com/Flussen/swagger-fiber-v3"
	"github.com/bronystylecrazy/ultrastructure/web"
)

type Middleware struct {
	path string
}

func (m *Middleware) Handle(r web.Router) {
	r.Get(m.path, swagger.HandlerDefault)
}
