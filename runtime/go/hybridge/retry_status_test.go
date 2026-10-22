//go:build amd64 || arm64

package hybridge

import (
	"fmt"
	"runtime"
	"testing"
	"unsafe"

	"hybrid.program/hybridge/internal/retryprobe"
)

//go:noinline
func growRetryProbeStack(depth int) byte {
	var frame [1024]byte
	frame[depth%len(frame)] = byte(depth)
	var child byte
	if depth > 0 {
		child = growRetryProbeStack(depth - 1)
	}
	runtime.KeepAlive(&frame)
	return child ^ frame[depth%len(frame)]
}

func TestNativeReturnValueDoesNotRetryCompletedCall(t *testing.T) {
	fn := retryprobe.Address()
	cases := []struct {
		name string
		call func(uintptr) uint64
	}{
		{"Cross1", func(p uintptr) uint64 { return Cross1(fn, p) }},
		{"Cross2", func(p uintptr) uint64 { return Cross2(fn, p, 2) }},
		{"Cross3", func(p uintptr) uint64 { return Cross3(fn, p, 2, 3) }},
		{"Cross4", func(p uintptr) uint64 { return Cross4(fn, p, 2, 3, 4) }},
		{"Cross1S", func(p uintptr) uint64 { return Cross1S(fn, p) }},
		{"Cross2S", func(p uintptr) uint64 { return Cross2S(fn, p, 2) }},
		{"Cross3S", func(p uintptr) uint64 { return Cross3S(fn, p, 2, 3) }},
		{"Cross4S", func(p uintptr) uint64 { return Cross4S(fn, p, 2, 3, 4) }},
	}
	for _, tc := range cases {
		for _, value := range []uint64{0, 42, ^uint64(0), ^uint64(1)} {
			t.Run(fmt.Sprintf("%s/return_%x", tc.name, value), func(t *testing.T) {
				state := &struct{ calls, value uint64 }{value: value}
				var pin runtime.Pinner
				pin.Pin(state)
				defer pin.Unpin()
				// Establish ample real headroom. Otherwise the legitimate deep
				// path could hide a value-sentinel collision in the fast path.
				growRetryProbeStack(64)
				if headroom := SPHeadroomForTest(); headroom < 2*fastStackNeed {
					t.Fatalf("probe needs real fast-path headroom, got %d", headroom)
				}
				before := DeepPathCount.Load()
				got := tc.call(uintptr(unsafe.Pointer(state)))
				runtime.KeepAlive(state)
				if got != value {
					t.Errorf("native return = %#x, want %#x", got, value)
				}
				if state.calls != 1 {
					t.Errorf("native callee ran %d times, want exactly once", state.calls)
				}
				if delta := DeepPathCount.Load() - before; delta != 0 {
					t.Errorf("completed fast-path call triggered %d deep retries", delta)
				}
			})
		}
	}
}
