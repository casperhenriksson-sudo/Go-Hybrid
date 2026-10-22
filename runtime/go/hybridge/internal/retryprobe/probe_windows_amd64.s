//go:build windows && amd64

#include "textflag.h"

TEXT ·Address(SB), NOSPLIT, $0-8
        LEAQ    ·invokeOnce(SB), AX
        MOVQ    AX, ret+0(FP)
        RET

// Win64: the first native argument is in CX.
TEXT ·invokeOnce(SB), NOSPLIT|NOFRAME, $0-0
        INCQ    0(CX)
        MOVQ    8(CX), AX
        RET
