package maestro

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// Wrapper around fleetctl, able to run every command. It uses two channels to communicate
// output and return code of every command issued.
func FleetExec(args []string, output chan string, exit chan int) {
	var (
		waitStatus syscall.WaitStatus
		cmdOut     bytes.Buffer
		cmdErr     bytes.Buffer
		out        string
	)
	exitCode := -1
	fleetArgs := []string{"--endpoint", flagFleetEndpoints, "--strict-host-key-checking=false"}
	fleetArgs = append(fleetArgs, flagFleetOptions...)
	fleetArgs = append(fleetArgs, args...)
	cmd := exec.Command("fleetctl", fleetArgs...)
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdErr
	if strings.HasPrefix(args[0], "list-") {
		Print(fmt.Sprintf("running %s", strings.Join(cmd.Args, " ")))
	} else {
		PrintD(fmt.Sprintf("running %s", strings.Join(cmd.Args, " ")))
	}
	if err := cmd.Run(); err != nil {
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

// Process output and exit channel from a fleetctl command.
func FleetProcessOutput(output chan string, exit chan int, trim ...bool) int {
	var exitCode int
	t := true
	if len(trim) > 0 {
		t = trim[0]
	}
	for out := range output {
		if out != "" {
			if t {
				out = strings.Trim(out, "\n")
			}
			Print(out)
		}
		exitCode += <-exit
	}
	return exitCode
}

// Utility function to check if a unit is already running on the cluster.
func FleetIsUnitRunning(unitPath string) (ret bool) {
	ret = false
	exitCode := FleetExecCommand("status", unitPath)
	if exitCode == 0 {
		ret = true
	}
	return
}

func FleetExecCommand(cmd, unitPath string) (exitCode int) {
	FleetCheckPath(unitPath)
	output := make(chan string)
	exit := make(chan int)
	go FleetExec([]string{cmd, unitPath}, output, exit)
	exitCode += FleetProcessOutput(output, exit)
	return
}

func FleetCheckPath(unitPath string) {
	if strings.Contains(unitPath, "@") {
		split := strings.Split(unitPath, "@")
		unitPath = fmt.Sprintf("%s@.service", split[0])
	}
	if _, err := os.Stat(unitPath); err != nil {
		PrintF(errors.New("invalid unit or maybe you forgot to run maestro build..."))
	}
}

func FleetBuildUnit(_, unitPath string) (exitCode int) {
	cmds := []string{"destroy", "submit", "load", "start"}
	for _, cmd := range cmds {
		exitCode += FleetExecCommand(cmd, unitPath)
	}
	return
}

func FleetRunUnit(_, unitPath string) (exitCode int) {
	cmds := []string{"submit", "load", "start"}
	if !FleetIsUnitRunning(unitPath) {
		for _, cmd := range cmds {
			exitCode += FleetExecCommand(cmd, unitPath)
		}
	}
	return
}
