package web

import "reflect"

func resetModuleStartRegistries(reg *RegistryContainer) {
	if reg == nil || reg.Metadata == nil {
		return
	}
	// Route metadata is runtime state and should always start clean per instance.
	reg.Metadata.Clear()
}

func resetModuleStopRegistries(reg *RegistryContainer) {
	if reg == nil {
		return
	}
	if reg.Metadata != nil {
		reg.Metadata.Clear()
	}
	if reg.Enum != nil {
		reg.Enum.mu.Lock()
		reg.Enum.values = make(map[reflect.Type][]interface{})
		reg.Enum.mu.Unlock()
	}
	if reg.SchemaName != nil {
		reg.SchemaName.mu.Lock()
		reg.SchemaName.names = make(map[reflect.Type]string)
		reg.SchemaName.mu.Unlock()
	}
	if reg.TypeRules != nil {
		reg.TypeRules.mu.Lock()
		reg.TypeRules.replace = make(map[reflect.Type]reflect.Type)
		reg.TypeRules.skipped = make(map[reflect.Type]struct{})
		reg.TypeRules.mu.Unlock()
	}
	if reg.Hook != nil {
		reg.Hook.mu.Lock()
		reg.Hook.hook = nil
		reg.Hook.mu.Unlock()
	}
	if reg.OperationIDHook != nil {
		reg.OperationIDHook.mu.Lock()
		reg.OperationIDHook.hook = nil
		reg.OperationIDHook.mu.Unlock()
	}
	if reg.OperationIDMode != nil {
		reg.OperationIDMode.mu.Lock()
		reg.OperationIDMode.enabled = false
		reg.OperationIDMode.sep = "_"
		reg.OperationIDMode.mu.Unlock()
	}
}
