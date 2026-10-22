#include "textflag.h"

TEXT ·Address(SB), NOSPLIT, $0-8
        MOVD    $·invokeOnce(SB), R0
        MOVD    R0, ret+0(FP)
        RET

// AAPCS64: the first native argument is in R0.
TEXT ·invokeOnce(SB), NOSPLIT|NOFRAME, $0-0
        MOVD    0(R0), R1
        ADD     $1, R1
        MOVD    R1, 0(R0)
        MOVD    8(R0), R0
        RET
