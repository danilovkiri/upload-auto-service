// Package syncutils provides synchronization objects for an app-wide usage.

package syncutils

import (
	"context"
	"sync"
)

// SyncUtils defines a new object and sets its attributes.
type SyncUtils struct {
	Wg         *sync.WaitGroup
	Ctx        context.Context
	SyncCancel context.CancelFunc
}

// NewSyncUtils initializes a new SyncUtils object.
func NewSyncUtils() *SyncUtils {
	ctx, cancel := context.WithCancel(context.Background())
	return &SyncUtils{
		Wg:         &sync.WaitGroup{},
		Ctx:        ctx,
		SyncCancel: cancel,
	}
}
