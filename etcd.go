package maestro

import (
	"errors"
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
	lg.Debug("checking if etcdctl is in your $PATH", etcdctl)
	_, err := exec.LookPath(etcdctl)
	if err != nil {
		lg.Error(errors.New(err.Error() + ". this is not fatal, you can still use fleetctl via ssh"))
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
	lg.Debug("etcdctl args "+strings.Join(cmd.Args, " "), etcdctl)
	exitCode = MaestroCommandExec(cmd, output)
	lg.Debug("exit code: "+strconv.Itoa(exitCode), etcdctl)
	exit <- exitCode
	close(exit)
	return
}

// Pulls maestro related keys
func EtcdPullKeys(skydns, all bool, key string) (exitCode int) {
	output := make(chan string)
	exit := make(chan int)
	args := EtcdPrepareArgs(key)
	lg.Out(lg.b("maestro ") + "running fleetctl" + strings.Join(args, " "))
	go EtcdExec(args, output, exit, key)
	for line := range output {
		line = string(line)
		if key != "" {
			lg.Out(strings.Trim(line, "\n"))
		} else {
			if !all {
				if strings.HasPrefix(line, "/maestro.io") {
					lg.Out(line)
				}
				if skydns {
					if strings.HasPrefix(line, "/skydns") {
						lg.Out(line)
					}
				}
			} else {
				if line != "" {
					lg.Out(line)
				}
			}
		}
	}
	exitCode = <-exit
	return
}
