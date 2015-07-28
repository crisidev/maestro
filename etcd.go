package maestro

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
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

// Prepares etcdctl arguments with info based from command line
func EtcdPrepareArgs(key string) (etcdArgs []string) {
	if key == "" {
		etcdArgs = []string{"ls", "--recursive", "--sort"}
	} else {
		etcdArgs = []string{"get", key}
	}
	etcdArgs = append([]string{"-C", etcdEndpoints}, etcdArgs...)
	return
}

// Wrapper around etcdctl, able to run every command. It uses two channels to communicate
// output and return code of every command issued.
func EtcdExec(args []string, output chan string, exit chan int, key string) {
	var exitCode int
	etcdArgs := EtcdPrepareArgs(key)
	cmd := exec.Command(etcdctl, etcdArgs...)
	PrintD(fmt.Sprintf("running %s", strings.Join(cmd.Args, " ")))
	exitCode = MaestroExec(cmd, output)
	PrintD("exit code: " + strconv.Itoa(exitCode))
	exit <- exitCode
	close(exit)
	return
}

// Pulls maestro related keys
func EtcdPullKeys(skydns, all bool, key string) (exitCode int) {
	output := make(chan string)
	exit := make(chan int)
	args := EtcdPrepareArgs(key)
	go EtcdExec(args, output, exit, key)
	for line := range output {
		line = string(line)
		if key != "" {
			Print(strings.Trim(line, "\n"))
		} else {
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
	}
	exitCode = <-exit
	return
}
