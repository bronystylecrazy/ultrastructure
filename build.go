package us

import "fmt"

type BuildInfo struct {
	Version string // Version is the version of the application.
	Commit  string // Commit is the commit hash of the application.
	Date    string // Date is the date of the build.
}

func (s BuildInfo) String() string {
	return fmt.Sprintf("Version: %s, Commit: %s, Date: %s", s.Version, s.Commit, s.Date)
}
