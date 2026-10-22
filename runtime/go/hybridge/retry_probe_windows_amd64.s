//go:build windows && amd64

#include "textflag.h"

TEXT ·retryProbeAddress(SB), NOSPLIT, $0-8
        LEAQ    ·retryProbeInvoke(SB), AX
        MOVQ    AX, ret+0(FP)
        RET

// Win64: the first native argument is in CX.
TEXT ·retryProbeInvoke(SB), NOSPLIT|NOFRAME, $0-0
        INCQ    0(CX)
        MOVQ    8(CX), AX
        RET
