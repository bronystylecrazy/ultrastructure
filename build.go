package us

import (
	"strings"

	"github.com/bronystylecrazy/ultrastructure/meta"
)

const (
	NilVersion   = meta.NilVersion
	NilCommit    = meta.NilCommit
	NilBuildDate = meta.NilBuildDate
)

func IsProduction() bool {
	version := strings.TrimSpace(meta.Version)
	return version != "" && version != NilVersion
}

func IsDevelopment() bool {
	return !IsProduction()
}
