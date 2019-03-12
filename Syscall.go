package KJUtil

import "syscall"

func Fork() (uintptr, syscall.Errno) {
	var ret uintptr
	var err syscall.Errno
	ret, _, err = syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	return ret, err
}
