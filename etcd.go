package maestro

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

const (
	etcdctl       = "etcdctl"
	etcdEndpoints = "172.17.8.103:2379,172.17.8.103:2379,172.17.8.103:2379"
)

// Checks if etcdctl is available on the system.
func EtcdCheckExec() {
	_, err := exec.LookPath(etcdctl)
	if err != nil {
		PrintE(errors.New(err.Error() + ". this is not fatal, you can still use fleetctl via ssh"))
	}
}

// Wrapper around fleetctl, able to run every command. It uses two channels to communicate
// output and return code of every command issued.
func EtcdExec(args []string, output chan string, exit chan int) {
	var (
		waitStatus syscall.WaitStatus
		cmdOut     bytes.Buffer
		cmdErr     bytes.Buffer
		out        string
	)
	exitCode := -1
	etcdArgs := []string{"-C", etcdEndpoints}
	etcdArgs = append(etcdArgs, args...)
	cmd := exec.Command(etcdctl, etcdArgs...)
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdErr
	PrintD(fmt.Sprintf("running %s", strings.Join(cmd.Args, " ")))
	if err := cmd.Run(); err != nil {
		PrintE(err)
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus = exitError.Sys().(syscall.WaitStatus)
			exitCode = waitStatus.ExitStatus()
		}
		out = string(cmdErr.Bytes())

	} else {
		// Success
		waitStatus = cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = waitStatus.ExitStatus()
	}
	out += string(cmdOut.Bytes())
	PrintD("output: " + out)
	PrintD("exit code: " + strconv.Itoa(exitCode))
	output <- out
	exit <- exitCode
	close(output)
	close(exit)
	return
}

// Pulls maestro related keys
func EtcdPullKeys(skydns, all bool, key string) (exitCode int) {
	output := make(chan string)
	exit := make(chan int)
	if key == "" {
		args := []string{"ls", "--recursive", "--sort"}
		go EtcdExec(args, output, exit)
		out := <-output
		exitCode = <-exit
		for _, line := range strings.Split(out, "\n") {
			if !all {
				if strings.HasPrefix(line, "/maestro.io") {
					Print(line)
				}
				if skydns {
					if strings.HasPrefix(line, "/skydns") {
						Print(line)
					}
				}
			} else {
				if line != "" {
					Print(line)
				}
			}
		}
	} else {
		args := []string{"get", key}
		go EtcdExec(args, output, exit)
		out := <-output
		exitCode += <-exit
		Print(strings.Trim(out, "\n"))
	}
	return
}
