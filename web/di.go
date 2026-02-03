package web

import "github.com/bronystylecrazy/ultrastructure/di"

func UseSwagger() di.Node {
	return di.Provide(NewSwaggerHandler)
}
