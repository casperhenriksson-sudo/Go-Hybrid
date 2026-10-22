//go:build amd64 || arm64

package hybridge

import "unsafe"

// retryProbeAddress returns a native function whose first argument points to two uint64
// words: it increments the first word and returns the second. Extra arguments
// are ignored, so the same callee exercises every supported bridge arity.
// These private fixture entrypoints are referenced only by tests and discarded
// from production binaries by the linker. Keep them flat with the bridge:
// hyb's runtime vendoring and generated workspaces copy package files directly.
func retryProbeAddress() unsafe.Pointer

// retryProbeInvoke is entered through the native ABI, never as a Go function.
func retryProbeInvoke()
