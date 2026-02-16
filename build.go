package us

import "github.com/bronystylecrazy/ultrastructure/meta"

const (
	NilVersion   = meta.NilVersion
	NilCommit    = meta.NilCommit
	NilBuildDate = meta.NilBuildDate
)

var (
	Name        string = meta.Name
	Description string = meta.Description

	Version   string = NilVersion
	Commit    string = NilCommit
	BuildDate string = NilBuildDate
)

func IsProduction() bool {
	return Version != NilVersion
}

func IsDevelopment() bool {
	return Version == NilVersion
}

func syncMeta() {
	meta.Name = Name
	meta.Description = Description
	meta.Version = Version
	meta.Commit = Commit
	meta.BuildDate = BuildDate
}

func init() {
	syncMeta()
}
