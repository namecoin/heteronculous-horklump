// Copyright 2022 Namecoin Developers.

// This file is part of heteronculous-horklump.
//
// heteronculous-horklump is free software: you can redistribute it and/or
// modify it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// heteronculous-horklump is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with heteronculous-horklump.  If not, see
// <https://www.gnu.org/licenses/>.

package main

import (
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/hlandau/dexlogconfig"
	"github.com/hlandau/xlog"
	"github.com/u-root/u-root/pkg/strace"
	"golang.org/x/sys/unix"
	"gopkg.in/hlandau/easyconfig.v1"
)

var log, _ = xlog.New("horklump")

type Config struct {
	Program  string   `usage:"Program Name"`
	SocksTCP string   `default:"127.0.0.1:9050"`
	Args     []string `usage:"Program Arguments"`
	KillProg string   `default:"n" usage:"Kill the Program in case of a Proxy Leak (y or n)"`
	LogLeaks string   `default:"n" usage:"Allow Proxy Leaks but Log any that Occur (y or n)"`
	EnvVar   string   `default:"y" usage:"Use the Environment Vars TOR_SOCKS_HOST and TOR_SOCKS_PORT (y or n)"`
}

func main() {
	cfg := Config{}
	config := easyconfig.Configurator{
		ProgramName: "horklump",
	}

	config.ParseFatal(&cfg)
	dexlogconfig.Init()
	program := exec.Command(cfg.Program, cfg.Args...) //nolint
	program.Stdin, program.Stdout, program.Stderr = os.Stdin, os.Stdout, os.Stderr

	if strings.ToLower(cfg.EnvVar) == "y" {
		cfg.SocksTCP = SetEnv(cfg.SocksTCP, os.Getenv("TOR_SOCKS_HOST"), os.Getenv("TOR_SOCKS_PORT"))
	}

	// Start the program with tracing.
	if err := strace.Trace(program, func(t strace.Task, record *strace.TraceRecord) error {
		if record.Event == strace.SyscallEnter && record.Syscall.Sysno == unix.SYS_CONNECT {
			if err := HandleConnect(t, record, program, cfg); err != nil {
				panic(err)
			}
		}
		return nil
	}); err != nil {
		panic(err)
	}
}

func HandleConnect(task strace.Task, record *strace.TraceRecord, program *exec.Cmd, cfg Config) error {
	data := strace.SysCallEnter(task, record.Syscall)
	// Detect the IP and Port.
	ip, port := GetIPAndPortdata(data, task, record.Syscall.Args)
	IPPort := fmt.Sprintf("%s:%s", ip, port)
	if IPPort == cfg.SocksTCP || ip == "/var/run/nscd/socket" { //nolint
		fmt.Printf("Connecting to %v\n", IPPort) //nolint
	} else {
		if strings.ToLower(cfg.LogLeaks) == "y" {
			log.Warnf("Proxy Leak detected, but allowed : %v", IPPort)
			return nil
		}
		if strings.ToLower(cfg.KillProg) == "y" {
			KillApp(program, IPPort)
			return nil
		}
		if err := syscall.PtraceSyscall(record.PID, 0); err != nil {
			return err
		}
		var status unix.WaitStatus
		if _, err := unix.Wait4(record.PID, &status, 0, nil); err != nil {
			return err
		}

		regs := &unix.PtraceRegs{}
		if err := unix.PtraceGetRegs(record.PID, regs); err != nil {
			return err
		}
		// set to invalid syscall
		regs.Rax = math.MaxUint64
		if err := unix.PtraceSetRegs(record.PID, regs); err != nil {
			return err
		}
		if err := syscall.PtraceSyscall(record.PID, 0); err != nil {
			return err
		}

		if _, err := unix.Wait4(record.PID, &status, 0, nil); err != nil {
			return err
		}

		fmt.Printf("Blocking -> %v\n", IPPort) //nolint
	}

	return nil
}

// SocketSysCalls checks if a syscall is a socket syscall.
func SocketSysCalls(r *strace.TraceRecord) error { //nolint
	// Socket call functions from Ubuntu Manuals (https://manpages.ubuntu.com/manpages/bionic/man2/socketcall.2.html)
	socketfunctions := map[string]struct{}{
		"socket": {}, "bind": {}, "connect": {}, "listen": {}, "accept": {}, "getsockname": {},
		"getpeername": {}, "socketpair": {}, "send": {}, "recv": {}, "sendto": {}, "recvfrom": {}, "shutdown": {}, "setsockopt": {},
		"getsockopt": {}, "sendmsg": {}, "recvmsg": {}, "accept4": {}, "recvmmsg": {}, "sendmmsg": {},
	}

	// Get the name of the Socket System Call
	SyscallName, _ := strace.ByNumber(uintptr(r.Syscall.Sysno))
	// Check if it's a Socket System Call
	if _, err := socketfunctions[SyscallName]; !err {
		return nil
	}
	fmt.Printf("Detected a Socket System Call: %v\n", SyscallName) //nolint

	return nil
}

func GetIPAndPortdata(data string, t strace.Task, args strace.SyscallArguments) (ip string, port string) { //nolint
	if len(data) == 0 {
		return
	}
	//  For the time being, the string slicing method is being used to extract the Address.
	s1 := strings.Index(data, "Addr:")
	if s1 != -1 {
		s2 := strings.Index(data[s1:], "}")
		s3 := strings.Index(data[s1:], ",")

		if s2 < s3 {
			ip = data[s1+5 : s1+s2]
		} else {
			ip = data[s1+5 : s1+s3]
		}

		ip = strings.ReplaceAll(ip, `"`, "")
		ip = strings.ReplaceAll(ip, ` `, "")

		if ip[:2] == "0x" {
			ip = ip[2:]
			// Decode the Address
			a, _ := hex.DecodeString(ip)
			ip = fmt.Sprintf("%v.%v.%v.%v", a[0], a[1], a[2], a[3])
		}
	}
	// To extract the Port, we use the functions - CaptureAddress and GetAddress.
	addr := args[1].Pointer()
	addrlen := args[2].Uint()

	socketaddr, err := strace.CaptureAddress(t, addr, addrlen)
	if err != nil {
		return "", ""
	}

	fulladdr, err := strace.GetAddress(t, socketaddr)
	if err != nil {
		return "", ""
	}

	P := fulladdr.Port
	port = strconv.Itoa(int(P))

	return ip, port
}

func KillApp(program *exec.Cmd, iPPort string) {
	err := program.Process.Signal(syscall.SIGKILL)
	if err != nil {
		fmt.Println("Failed to kill the application: %v\n", err) //nolint
		panic(err)
	}
	fmt.Printf("Proxy Leak Detected : %v. Killing the Application.\n", iPPort) //nolint
}

func SetEnv(socks string, host string, port string) string {
	tcpsocks := strings.Split(socks, ":")

	switch {
	case (host == "" && port != ""):
		return tcpsocks[0] + ":" + port
	case (host != "" && port == ""):
		return host + ":" + tcpsocks[1]
	case (host != "" && port != ""):
		return host + ":" + port
	default:
		return socks
	}
}
