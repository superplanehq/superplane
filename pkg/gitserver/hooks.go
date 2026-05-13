package gitserver

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

// PostPublishHook is a global hook that gets called after canvas version publish.
// The gitserver package registers a callback here.

var (
	postPublishHooksMu sync.RWMutex
	postPublishHooks   []func(canvasID, orgID, userName string)
)

// RegisterPostPublishHook adds a callback that fires after canvas version publish.
func RegisterPostPublishHook(fn func(canvasID, orgID, userName string)) {
	postPublishHooksMu.Lock()
	defer postPublishHooksMu.Unlock()
	postPublishHooks = append(postPublishHooks, fn)
	log.Info("gitserver: registered post-publish hook for reverse sync")
}

// FirePostPublishHooks is called from the publish code path.
func FirePostPublishHooks(canvasID, orgID, userName string) {
	postPublishHooksMu.RLock()
	defer postPublishHooksMu.RUnlock()

	for _, fn := range postPublishHooks {
		fn(canvasID, orgID, userName)
	}
}

var (
	postCreateHooksMu sync.RWMutex
	postCreateHooks   []func(canvasID, orgID, canvasName string)
)

// RegisterPostCreateHook adds a callback that fires after canvas creation.
func RegisterPostCreateHook(fn func(canvasID, orgID, canvasName string)) {
	postCreateHooksMu.Lock()
	defer postCreateHooksMu.Unlock()
	postCreateHooks = append(postCreateHooks, fn)
	log.Info("gitserver: registered post-create hook for auto-init repos")
}

// FirePostCreateHooks is called from the canvas creation code path.
func FirePostCreateHooks(canvasID, orgID, canvasName string) {
	postCreateHooksMu.RLock()
	defer postCreateHooksMu.RUnlock()

	for _, fn := range postCreateHooks {
		fn(canvasID, orgID, canvasName)
	}
}
