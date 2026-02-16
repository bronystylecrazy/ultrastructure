package meta

const (
	NilVersion   = "v0.0.0-development"
	NilCommit    = "unknown"
	NilBuildDate = "unknown"
)

var (
	Name        string = "Ultrastructure"
	Description string = "a lightweight web framework for Go based on UberFX."

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
