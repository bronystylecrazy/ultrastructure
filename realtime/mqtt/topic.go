package mqtt

type TopicRegistrar interface {
	Topic(filter string, args ...any) error
}

type TopicSubscriber interface {
	Subscribe(r TopicRegistrar) error
}
