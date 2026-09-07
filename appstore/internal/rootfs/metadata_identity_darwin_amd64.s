#include "textflag.h"

TEXT libc_acl_get_fd_np_identity_trampoline<>(SB),NOSPLIT,$0-0
	JMP libc_acl_get_fd_np_identity(SB)
GLOBL ·libcACLGetFdNpIdentityTrampolineAddr(SB), RODATA, $8
DATA ·libcACLGetFdNpIdentityTrampolineAddr(SB)/8, $libc_acl_get_fd_np_identity_trampoline<>(SB)

TEXT libc_acl_free_identity_trampoline<>(SB),NOSPLIT,$0-0
	JMP libc_acl_free_identity(SB)
GLOBL ·libcACLFreeIdentityTrampolineAddr(SB), RODATA, $8
DATA ·libcACLFreeIdentityTrampolineAddr(SB)/8, $libc_acl_free_identity_trampoline<>(SB)

TEXT libc_acl_size_identity_trampoline<>(SB),NOSPLIT,$0-0
	JMP libc_acl_size_identity(SB)
GLOBL ·libcACLSizeIdentityTrampolineAddr(SB), RODATA, $8
DATA ·libcACLSizeIdentityTrampolineAddr(SB)/8, $libc_acl_size_identity_trampoline<>(SB)

TEXT libc_acl_copy_ext_identity_trampoline<>(SB),NOSPLIT,$0-0
	JMP libc_acl_copy_ext_identity(SB)
GLOBL ·libcACLCopyExtIdentityTrampolineAddr(SB), RODATA, $8
DATA ·libcACLCopyExtIdentityTrampolineAddr(SB)/8, $libc_acl_copy_ext_identity_trampoline<>(SB)
