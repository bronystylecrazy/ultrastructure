package us

var ConfigDisableTunnel string = "disable_tunnel"

type Configer interface {
	Get(key any, def any) error
	Set(key any, value any) error
}
