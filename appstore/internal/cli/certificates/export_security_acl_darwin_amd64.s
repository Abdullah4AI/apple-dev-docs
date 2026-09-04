#include "textflag.h"

TEXT libc_acl_get_fd_np_trampoline<>(SB),NOSPLIT,$0-0
	JMP	libc_acl_get_fd_np(SB)
GLOBL	·libcACLGetFdNpTrampolineAddr(SB), RODATA, $8
DATA	·libcACLGetFdNpTrampolineAddr(SB)/8, $libc_acl_get_fd_np_trampoline<>(SB)

TEXT libc_acl_free_trampoline<>(SB),NOSPLIT,$0-0
	JMP	libc_acl_free(SB)
GLOBL	·libcACLFreeTrampolineAddr(SB), RODATA, $8
DATA	·libcACLFreeTrampolineAddr(SB)/8, $libc_acl_free_trampoline<>(SB)

TEXT libc_acl_init_trampoline<>(SB),NOSPLIT,$0-0
	JMP	libc_acl_init(SB)
GLOBL	·libcACLInitTrampolineAddr(SB), RODATA, $8
DATA	·libcACLInitTrampolineAddr(SB)/8, $libc_acl_init_trampoline<>(SB)

TEXT libc_acl_set_fd_np_trampoline<>(SB),NOSPLIT,$0-0
	JMP	libc_acl_set_fd_np(SB)
GLOBL	·libcACLSetFdNpTrampolineAddr(SB), RODATA, $8
DATA	·libcACLSetFdNpTrampolineAddr(SB)/8, $libc_acl_set_fd_np_trampoline<>(SB)
