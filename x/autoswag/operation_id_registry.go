package autoswag

import (
	"regexp"
	"strings"
	"sync"
)

var nonWordOperationIDChars = regexp.MustCompile(`\W+`)

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

func RegisterOperationIDTagPrefix(separator string) {
	if separator == "" {
		separator = "_"
	}
	operationIDRegistry.mu.Lock()
	defer operationIDRegistry.mu.Unlock()
	operationIDRegistry.enabled = true
	operationIDRegistry.sep = separator
}

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

func sanitizeOperationIDPart(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	return nonWordOperationIDChars.ReplaceAllString(v, "_")
}
