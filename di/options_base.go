package di

// Option is applied to providers and/or invocations.
type Option interface {
	applyBind(*bindConfig)
	applyParam(*paramConfig)
}
