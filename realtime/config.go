package realtime

type TCPListenerConfig struct {
	ID      string `mapstructure:"id" default:"t1"`
	Address string `mapstructure:"address" default:":1883"`
}

type WebsocketListenerConfig struct {
	ID   string `mapstructure:"id" default:"ws1"`
	Path string `mapstructure:"path" default:"/realtime"`
}

type TopicACLConfig struct {
	Enabled         bool     `mapstructure:"enabled" default:"false"`
	AllowedPrefixes []string `mapstructure:"allowed_prefixes"`
}

type Config struct {
	TCPListener       TCPListenerConfig       `mapstructure:"tcp_listener"`
	WebsocketListener WebsocketListenerConfig `mapstructure:"websocket_listener"`
	TopicACL          TopicACLConfig          `mapstructure:"topic_acl"`
}
