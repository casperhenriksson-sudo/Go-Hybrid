//go:build amd64 && !windows

#include "textflag.h"

TEXT ·retryProbeAddress(SB), NOSPLIT, $0-8
        LEAQ    ·retryProbeInvoke(SB), AX
        MOVQ    AX, ret+0(FP)
        RET

// SysV: the first native argument is in DI.
TEXT ·retryProbeInvoke(SB), NOSPLIT|NOFRAME, $0-0
        INCQ    0(DI)
        MOVQ    8(DI), AX
        RET
