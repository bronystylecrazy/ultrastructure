package realtime

type TCPListenerConfig struct {
	ID      string `mapstructure:"id" default:"t1"`
	Address string `mapstructure:"address" default:":1883"`
}

type WebsocketListenerConfig struct {
	ID   string `mapstructure:"id" default:"ws1"`
	Path string `mapstructure:"path" default:"/realtime"`
}

type Config struct {
	TCPListener       TCPListenerConfig       `mapstructure:"tcp_listener"`
	WebsocketListener WebsocketListenerConfig `mapstructure:"websocket_listener"`
}
