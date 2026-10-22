//go:build amd64 || arm64

// Package retryprobe supplies a tiny native-ABI callee for bridge tests.
// Only tests import it; production bridge callers do not link this fixture.
package retryprobe

import "unsafe"

// Address returns a native function whose first argument points to two uint64
// words: it increments the first word and returns the second. Extra arguments
// are ignored, so the same callee exercises every supported bridge arity.
func Address() unsafe.Pointer

// invokeOnce is entered through the native ABI, never as a Go function.
func invokeOnce()
