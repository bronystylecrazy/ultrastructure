package realtime

type TopicRegistrar interface {
	Use(middlewares ...TopicMiddleware)
	Topic(filter string, args ...any) error
}

type TopicMiddleware func(next TopicHandler) TopicHandler
