package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"syscall"
)

type Process struct {
	SysProcAttr syscall.SysProcAttr
	Handler     func() int
}

var commands = map[string]Process{
	"net": Process{
		SysProcAttr: syscall.SysProcAttr{Cloneflags: syscall.CLONE_NEWNET},
		Handler:     displayInterfaces,
	},
	"pid": Process{
		SysProcAttr: syscall.SysProcAttr{Cloneflags: syscall.CLONE_NEWPID},
		Handler:     displayProcessID,
	},
	"uts": Process{
		SysProcAttr: syscall.SysProcAttr{Cloneflags: syscall.CLONE_NEWUTS},
		Handler:     displayHostname,
	},
	"mount": Process{
		SysProcAttr: syscall.SysProcAttr{Cloneflags: syscall.CLONE_NEWNS},
		Handler:     displayRootDirectory,
	},
}

func main() {
	if len(os.Args) > 1 {
		child()
		return
	}

	parent()
}

func parent() {
	for name, command := range commands {
		cmd := exec.Command(
			"/proc/self/exe",
			name,
		)

		fmt.Printf("Running /proc/self/exe %s\n", name)

		cmd.SysProcAttr = &command.SysProcAttr
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Printf("Error running subprocess '%s'\n", err)
			os.Exit(0)
		}
	}
}

func child() {
	ns := os.Args[1]
	cmd, ok := commands[ns]

	if !ok {
		fmt.Println("Unknown namespace to test")
		os.Exit(1)
	}

	os.Exit(cmd.Handler())
}

func displayHostname() int {
	err := syscall.Sethostname([]byte("container-hostname"))
	if err != nil {
		fmt.Println(err.Error())
		return 1
	}

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err.Error())
		return 1
	}

	fmt.Printf("Current hostname is %s\n", hostname)
	return 0
}

func displayProcessID() int {
	fmt.Printf("Current process ID is %d\n", os.Getpid())
	return 0
}

func displayInterfaces() int {
	ifaces, err := net.Interfaces()

	if err != nil {
		fmt.Println(err.Error())
		return 1
	}

	for _, iface := range ifaces {
		fmt.Printf("Found interface %s\n", iface.Name)
	}

	return 0
}

func displayMounts() int {
	rawMounts, err := ioutil.ReadFile("/proc/self/mounts")

	if err != nil {
		fmt.Println("Error displaying mount points")
		return 1
	}

	fmt.Println(string(rawMounts))

	return 0
}

func displayRootDirectory() int {
	fmt.Println("Before mounting:")

	if ret := displayMounts(); ret > 0 {
		return ret
	}

	err := os.Mkdir("/xxx", 0777)

	if err != nil {
		fmt.Println("Error creating directory")
		return 1
	}

	err = syscall.Mount("", "/xxx", "tmpfs", syscall.MS_ACTIVE, "")

	if err != nil {
		fmt.Println("Error mounting tmpfs")
		return 1
	}

	fmt.Println("After mounting:")

	if ret := displayMounts(); ret > 0 {
		return ret
	}

	err = syscall.Unmount("/xxx", syscall.MNT_DETACH)

	if err != nil {
		fmt.Println("Error unmounting tmpfs")
		return 1
	}

	err = os.Remove("/xxx")

	if err != nil {
		fmt.Println("Error removing directory")
		return 1
	}

	return 0
}
