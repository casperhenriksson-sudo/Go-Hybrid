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

//go:noinline
func retryProbeAtShortHeadroom(fn unsafe.Pointer, p uintptr, variant, depth int) (value uint64, retry bool, headroom uintptr) {
	var frame [256]byte
	frame[depth%len(frame)] = byte(depth)
	headroom = SPHeadroomForTest()
	// This interval is wider than this helper's frame and lies below both
	// real guards. Walk down the actual stack; never alter runtime offsets
	// or replace the headroom check with a fake resource measurement.
	if headroom >= 128 && headroom < 768 {
		switch variant {
		case 0:
			value, retry = callFast1(fn, p)
		case 1:
			value, retry = callFast2(fn, p, 2)
		case 2:
			value, retry = callFast3(fn, p, 2, 3)
		case 3:
			value, retry = callFast4(fn, p, 2, 3, 4)
		case 4:
			value, retry = callFast1S(fn, p)
		case 5:
			value, retry = callFast2S(fn, p, 2)
		case 6:
			value, retry = callFast3S(fn, p, 2, 3)
		case 7:
			value, retry = callFast4S(fn, p, 2, 3, 4)
		}
	} else if depth > 0 {
		value, retry, headroom = retryProbeAtShortHeadroom(fn, p, variant, depth-1)
	}
	runtime.KeepAlive(&frame)
	return
}

func TestShortHeadroomRequestsRetryWithoutInvokingNative(t *testing.T) {
	for variant := 0; variant < 8; variant++ {
		t.Run(fmt.Sprintf("variant_%d", variant), func(t *testing.T) {
			state := &struct{ calls, value uint64 }{value: ^uint64(1)}
			var pin runtime.Pinner
			pin.Pin(state)
			defer pin.Unpin()
			value, retry, headroom := retryProbeAtShortHeadroom(retryprobe.Address(), uintptr(unsafe.Pointer(state)), variant, 1024)
			runtime.KeepAlive(state)
			if headroom < 128 || headroom >= 768 {
				t.Fatalf("did not reach real short-stack probe interval: %d", headroom)
			}
			if !retry || value != 0 || state.calls != 0 {
				t.Fatalf("headroom=%d: value=%#x retry=%v nativeCalls=%d; want zero/true/zero", headroom, value, retry, state.calls)
			}
		})
	}
}
