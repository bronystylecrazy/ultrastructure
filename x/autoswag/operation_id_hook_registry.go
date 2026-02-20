package autoswag

import (
	"strings"
	"sync"
)

// OperationIDHookContext contains the current route context for operationId customization.
type OperationIDHookContext struct {
	Route       RouteInfo
	Metadata    *RouteMetadata
	Tags        []string
	OperationID string
	Generated   bool
}

// OperationIDHook allows custom global operationId transformation.
type OperationIDHook func(ctx OperationIDHookContext) string

type operationIDHookRegistryState struct {
	mu   sync.RWMutex
	hook OperationIDHook
}

func newOperationIDHookRegistryState() *operationIDHookRegistryState {
	return &operationIDHookRegistryState{}
}

var operationIDHookRegistry = newOperationIDHookRegistryState()

// RegisterOperationIDHook registers a global hook to customize operationId.
func RegisterOperationIDHook(hook OperationIDHook) {
	operationIDHookRegistry.mu.Lock()
	defer operationIDHookRegistry.mu.Unlock()
	operationIDHookRegistry.hook = hook
}

// ClearOperationIDHook clears the registered operationId hook.
func ClearOperationIDHook() {
	operationIDHookRegistry.mu.Lock()
	defer operationIDHookRegistry.mu.Unlock()
	operationIDHookRegistry.hook = nil
}

func applyRegisteredOperationIDHook(ctx OperationIDHookContext) string {
	operationIDHookRegistry.mu.RLock()
	hook := operationIDHookRegistry.hook
	operationIDHookRegistry.mu.RUnlock()
	if hook == nil {
		return ctx.OperationID
	}

	custom := strings.TrimSpace(hook(ctx))
	if custom == "" {
		return ctx.OperationID
	}
	return custom
}
