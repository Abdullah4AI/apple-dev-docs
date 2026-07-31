#include "textflag.h"

TEXT libc_fcopyfile_trampoline<>(SB),NOSPLIT,$0-0
	JMP	libc_fcopyfile(SB)
GLOBL	·libcFcopyfileTrampolineAddr(SB), RODATA, $8
DATA	·libcFcopyfileTrampolineAddr(SB)/8, $libc_fcopyfile_trampoline<>(SB)
