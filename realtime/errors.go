package realtime

import "errors"

var ErrTopicRegistrarStopped = errors.New("realtime: topic registrar is stopped")
var ErrTopicNotAllowed = errors.New("realtime: topic is not allowed by acl")
var ErrInvalidTopicRegistrationArgs = errors.New("realtime: invalid topic registration args")
var ErrTopicHandlerPanic = errors.New("realtime: panic in topic handler")
var ErrTopicHandlerTimeout = errors.New("realtime: topic handler timed out")
var ErrTopicCtxNoClient = errors.New("realtime: topic context has no client")
var ErrTopicCtxNoPublisher = errors.New("realtime: topic context has no publisher")
var ErrTopicCtxSessionControlUnsupported = errors.New("realtime: session control is unsupported")
var ErrTopicClientDisconnectedByHandler = errors.New("realtime: client disconnected by topic handler")
var ErrTopicClientRejectedByHandler = errors.New("realtime: client rejected by topic handler")
var ErrInvalidMessage = errors.New("message type not binary")
