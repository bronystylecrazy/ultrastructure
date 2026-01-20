package lifecycle

import "context"

type Starter interface {
	Start(context.Context) error
}

type Stopper interface {
	Stop(context.Context) error
}

type StartStoper interface {
	Starter
	Stopper
}
