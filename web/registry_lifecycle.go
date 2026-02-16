package web

import "context"

type registryLifecycle struct {
	reg *RegistryContainer
}

func NewRegistryLifecycle(reg *RegistryContainer) *registryLifecycle {
	return &registryLifecycle{reg: reg}
}

func (r *registryLifecycle) Start(context.Context) error {
	ActivateRegistryContainer(r.reg)
	resetModuleStartRegistries(r.reg)
	return nil
}

func (r *registryLifecycle) Stop(context.Context) error {
	resetModuleStopRegistries(r.reg)
	ResetDefaultRegistryContainer()
	return nil
}
