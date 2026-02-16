package web

import "sync"

type operationIDRegistryState struct {
	mu      sync.RWMutex
	enabled bool
	sep     string
}

func newOperationIDRegistryState() *operationIDRegistryState {
	return &operationIDRegistryState{
		sep: "_",
	}
}

var operationIDRegistry = newOperationIDRegistryState()

// RegisterOperationIDTagPrefix enables global operationId prefixing with first operation tag.
// Example: Users_GetUserByID.
func RegisterOperationIDTagPrefix(separator string) {
	if separator == "" {
		separator = "_"
	}
	operationIDRegistry.mu.Lock()
	defer operationIDRegistry.mu.Unlock()
	operationIDRegistry.enabled = true
	operationIDRegistry.sep = separator
}

// ClearOperationIDTagPrefix disables global operationId tag prefixing.
func ClearOperationIDTagPrefix() {
	operationIDRegistry.mu.Lock()
	defer operationIDRegistry.mu.Unlock()
	operationIDRegistry.enabled = false
	operationIDRegistry.sep = "_"
}

func getOperationIDTagPrefixConfig() (enabled bool, separator string) {
	operationIDRegistry.mu.RLock()
	defer operationIDRegistry.mu.RUnlock()
	return operationIDRegistry.enabled, operationIDRegistry.sep
}
