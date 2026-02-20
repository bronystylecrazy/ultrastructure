package web

import (
	"github.com/bronystylecrazy/ultrastructure/di"
)

func IgnoreAutoGroupHandlers() di.Option {
	return di.AutoGroupIgnoreType[Handler](HandlersGroupName)
}

func Init() di.Node {
	return di.Options(
		di.AutoGroup[Handler](HandlersGroupName),
		di.Invoke(SetupHandlers),
		di.Invoke(RegisterFiberApp),
	)
}
