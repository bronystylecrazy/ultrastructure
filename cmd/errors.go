package cmd

import "errors"

var ErrServiceControllerNotConfigured = errors.New("service controller is not configured")
var ErrServiceStatusNotSupported = errors.New("service status is not supported by controller")
