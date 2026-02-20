package autoswag

import "github.com/bronystylecrazy/ultrastructure/web"

var defaultRegistry = web.NewMetadataRegistry()

func GetGlobalRegistry() *MetadataRegistry {
	return defaultRegistry
}
