package di

// Option is applied to providers and/or invocations.
type Option interface {
	applyBind(*bindConfig)
	applyParam(*paramConfig)
}

// NodeOption is implemented by types that can act as both Node and Option.
type NodeOption interface {
	Node
	Option
}
