package main

import (
	"os/exec"
	"github.com/u-root/u-root/pkg/strace"
	"fmt"
)

func main() {
	// Start the program with tracing.
	if err := strace.Trace(exec.Command("ping","google.com"), func(t strace.Task, record *strace.TraceRecord) error{
		return SocketSysCalls(record)			
	}); err != nil {
		panic(err)
	}
}
// SocketSysCalls checks if a syscall is a socket syscall.
func SocketSysCalls(r *strace.TraceRecord) error{
	// Socket call functions from Ubuntu Manuals (https://manpages.ubuntu.com/manpages/bionic/man2/socketcall.2.html)
	socketfunctions := map[string]struct{}{"socket":{},"bind":{},"connect":{},"listen":{},"accept":{},"getsockname":{},
	"getpeername":{},"socketpair":{},"send":{},"recv":{},"sendto":{},"recvfrom":{},"shutdown":{},"setsockopt":{},
	"getsockopt":{},"sendmsg":{},"recvmsg":{},"accept4":{},"recvmmsg":{},"sendmmsg":{}}

	// Get the name of the Socket System Call
	SyscallName, _ := strace.ByNumber(uintptr(r.Syscall.Sysno))
	// Check if it's a Socket System Call
	if _, err := socketfunctions[SyscallName]; !err {
		return nil
	}
	fmt.Printf("Detected a Socket System Call: %v\n", SyscallName)
	return nil
}
 