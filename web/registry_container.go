package web

// RegistryContainer groups mutable runtime registries for web routing metadata.
// It is DI-provided and activated by web.Module lc.
type RegistryContainer struct {
	Metadata *MetadataRegistry
}

// NewRegistryContainer creates a fresh registry set.
func NewRegistryContainer() *RegistryContainer {
	return &RegistryContainer{
		Metadata: NewMetadataRegistry(),
	}
}
