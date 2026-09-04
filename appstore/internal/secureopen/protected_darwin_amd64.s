#include "textflag.h"

TEXT libc_fcntl_trampoline<>(SB),NOSPLIT,$0-0
	JMP	libc_fcntl(SB)
GLOBL	·libcFcntlTrampolineAddr(SB), RODATA, $8
DATA	·libcFcntlTrampolineAddr(SB)/8, $libc_fcntl_trampoline<>(SB)
