package maestro

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const fleetctl = "fleetctl"

// Checks if fleetctl is available on the system.
func FleetCheckExec() {
	lg.Debug("checking if fleetctl is in your $PATH", fleetctl)
	_, err := exec.LookPath(fleetctl)
	lg.Fatal(err)
}

// Prepares fleetctl arguments with info based from command line
func FleetPrepareArgs(args []string) (fleetArgs []string) {
	fleetArgs = []string{"--strict-host-key-checking=false"}
	if fleetEndpoints == "" {
		fleetArgs = append(fleetArgs, "--tunnel")
		fleetArgs = append(fleetArgs, fleetAddress)
	} else {
		fleetArgs = append(fleetArgs, "--endpoint")
		fleetArgs = append(fleetArgs, fleetEndpoints)
	}
	fleetArgs = append(fleetArgs, fleetOptions...)
	fleetArgs = append(fleetArgs, args...)
	lg.Debug("fleet args "+strings.Join(fleetArgs, " "), fleetctl)
	return
}

// Wrapper around fleetctl, able to run every command. It uses two channels to communicate
// output and return code of every command issued.
func FleetExec(args []string, output chan string, exit chan int) {
	var exitCode int
	fleetArgs := FleetPrepareArgs(args)
	cmd := exec.Command(fleetctl, fleetArgs...)
	exitCode = MaestroCommandExec(cmd, output)
	lg.Debug("exit code: "+strconv.Itoa(exitCode), fleetctl)
	exit <- exitCode
	close(exit)
	return
}

// Process output and exit channel from a fleetctl command.
func FleetProcessOutput(output chan string, exit chan int) (exitCode int) {
	for out := range output {
		lg.Out(out)
	}
	exitCode = <-exit
	return
}

// Utility function to check if a unit is already running on the cluster.
func FleetIsUnitRunning(unitPath string) (ret bool) {
	ret = false
	output := make(chan string)
	exit := make(chan int)
	go FleetExec([]string{"status", unitPath}, output, exit)
	_ = <-output
	exitCode := <-exit
	if exitCode == 0 {
		lg.Out("unit " + lg.b(unitPath) + " already running")
		ret = true
	} else if exitCode == 3 {
		lg.Out("unit " + lg.b(unitPath) + " already starting")
		ret = true
	}
	return
}

// Checks if a unit path is valid, either build unit and run unit.
func FleetCheckPath(unitPath string) {
	if strings.Contains(unitPath, "@") {
		split := strings.Split(unitPath, "@")
		unitPath = fmt.Sprintf("%s@.service", split[0])
	}
	if _, err := os.Stat(unitPath); err != nil {
		lg.Debug2("invalid unit or maybe you forgot to run ", "maestro build", fleetctl)
		lg.Fatal(err)
	}
	lg.Debug("unit "+unitPath+" is valid", fleetctl)
}

// Function able to run a command on a unit path. Output is processed and printed
// and fleetctl exit code is returned.
func FleetExecCommand(cmd, unitPath string) (exitCode int) {
	var args []string
	output := make(chan string)
	exit := make(chan int)
	FleetCheckPath(unitPath)
	args = []string{cmd}
	if cmd == "status" || strings.HasPrefix(cmd, "journal") {
		if strings.HasPrefix(cmd, "journal") {
			args = []string{"journal"}
			if cmd == "journalf" {
				args = append(args, "-f")
			} else if cmd == "journala" {
				args = append(args, "-lines=10000")
			}
		}
	}
	go FleetExec(append(args, unitPath), output, exit)
	if exitCode == 3 && (cmd == "status" || strings.HasPrefix(cmd, "journal")) {
		lg.Debug("please wait, unit " + unitPath + " is starting")
		exitCode = 0
	}
	exitCode += FleetProcessOutput(output, exit)
	return
}

// Wrapper to run a container build on the coreos cluster.
func FleetBuildUnit(_, unitPath string) (exitCode int) {
	lg.Debug("building "+unitPath+" on the cluser", fleetctl)
	cmds := []string{"destroy", "submit", "load", "start"}
	for _, cmd := range cmds {
		exitCode += FleetExecCommand(cmd, unitPath)
	}
	lg.Out("check results with " + lg.b("maestro buildstatus <unit name>"))
	return
}

// Wrapper to run a unit on the coreos cluster.
func FleetRunUnit(_, unitPath string) (exitCode int) {
	lg.Debug("running "+unitPath+" on the cluser", fleetctl)
	cmds := []string{"submit", "load", "start"}
	if !FleetIsUnitRunning(unitPath) {
		for _, cmd := range cmds {
			exitCode += FleetExecCommand(cmd, unitPath)
		}
	}
	return
}
