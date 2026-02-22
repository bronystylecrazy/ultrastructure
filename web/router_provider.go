package web

func NewModuleRouter(router *FiberServer, registries *RegistryContainer) Router {
	return NewRouterWithRegistry(router.App, registries.Metadata)
}
