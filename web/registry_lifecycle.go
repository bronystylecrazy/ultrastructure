package web

import "context"

type registryLifecycle struct {
	reg *RegistryContainer
}

func NewRegistryLifecycle(reg *RegistryContainer) *registryLifecycle {
	return &registryLifecycle{reg: reg}
}

func (r *registryLifecycle) Start(context.Context) error {
	if r.reg != nil && r.reg.Metadata != nil {
		r.reg.Metadata.Clear()
	}
	return nil
}

func (r *registryLifecycle) Stop(context.Context) error {
	if r.reg != nil && r.reg.Metadata != nil {
		r.reg.Metadata.Clear()
	}
	return nil
}
