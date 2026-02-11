//go:build !prod
// +build !prod

package us

const (
	NilVersion   = "v0.0.0-development"
	NilCommit    = "unknown"
	NilBuildDate = "unknown"
)

var (
	Version   string = NilVersion
	Commit    string = NilCommit
	BuildDate string = NilBuildDate
)
