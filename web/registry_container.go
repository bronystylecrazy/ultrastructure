package web

// RegistryContainer groups all mutable runtime registries used by autoswagger/web metadata.
// It is DI-provided and activated by web.Module lifecycle.
type RegistryContainer struct {
	Metadata        *MetadataRegistry
	Enum            *enumRegistryState
	SchemaName      *schemaNameRegistryState
	TypeRules       *typeRulesRegistryState
	Hook            *hookRegistryState
	OperationIDHook *operationIDHookRegistryState
	OperationIDMode *operationIDRegistryState
}

// NewRegistryContainer creates a fresh registry set.
func NewRegistryContainer() *RegistryContainer {
	return &RegistryContainer{
		Metadata:        newMetadataRegistry(),
		Enum:            newEnumRegistryState(),
		SchemaName:      newSchemaNameRegistryState(),
		TypeRules:       newTypeRulesRegistryState(),
		Hook:            newHookRegistryState(),
		OperationIDHook: newOperationIDHookRegistryState(),
		OperationIDMode: newOperationIDRegistryState(),
	}
}

// ActivateRegistryContainer swaps package-level registry pointers to this container.
func ActivateRegistryContainer(c *RegistryContainer) {
	if c == nil {
		return
	}
	if c.Metadata != nil {
		globalRegistry = c.Metadata
	}
	if c.Enum != nil {
		enumRegistry = c.Enum
	}
	if c.SchemaName != nil {
		schemaNameRegistry = c.SchemaName
	}
	if c.TypeRules != nil {
		typeRulesRegistry = c.TypeRules
	}
	if c.Hook != nil {
		hookRegistry = c.Hook
	}
	if c.OperationIDHook != nil {
		operationIDHookRegistry = c.OperationIDHook
	}
	if c.OperationIDMode != nil {
		operationIDRegistry = c.OperationIDMode
	}
}

// ResetDefaultRegistryContainer restores package-level registries to fresh defaults.
func ResetDefaultRegistryContainer() {
	ActivateRegistryContainer(NewRegistryContainer())
}
