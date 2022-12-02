package main

import (
	"os/exec"
	"github.com/u-root/u-root/pkg/strace"
	"fmt"
)

func main() {
	socketfunctions := map[string]struct{}{"socket":{},"bind":{},"connect":{},"listen":{},"accept":{},"getsockname":{},
	"getpeername":{},"socketpair":{},"send":{},"recv":{},"sendto":{},"recvfrom":{},"shutdown":{},"setsockopt":{},
	"getsockopt":{},"sendmsg":{},"recvmsg":{},"accept4":{},"recvmmsg":{},"sendmmsg":{}}
	
	if err := strace.Trace(exec.Command("ping","google.com"), func(t strace.Task, record *strace.TraceRecord) error {
		SyscallName, _ := strace.ByNumber(uintptr(record.Syscall.Sysno))
		if _, err := socketfunctions[SyscallName]; !err {
			return nil
		}

		fmt.Printf("Detect a Socket System Call: %v", SyscallName)
		return nil
	}); err != nil {
		panic(err)
	}
}
