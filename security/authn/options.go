package authn

type ErrorMode string

const (
	ErrorModeFailFast   ErrorMode = "fail_fast"
	ErrorModeBestEffort ErrorMode = "best_effort"
)
