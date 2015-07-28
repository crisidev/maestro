package maestro

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

const fleetctl = "fleetctl"

// Checks if fleetctl is available on the system.
func FleetCheckExec() {
	_, err := exec.LookPath(fleetctl)
	if err != nil {
		PrintF(err)
	}
}

func FleetPrepareCmd(args []string) (cmd *exec.Cmd) {
	fleetArgs := []string{"--strict-host-key-checking=false"}
	if fleetEndpoints == "" {
		fleetArgs = append(fleetArgs, "--tunnel")
		fleetArgs = append(fleetArgs, fleetAddress)
	} else {
		fleetArgs = append(fleetArgs, "--endpoint")
		fleetArgs = append(fleetArgs, fleetEndpoints)
	}
	fleetArgs = append(fleetArgs, fleetOptions...)
	fleetArgs = append(fleetArgs, args...)
	cmd = exec.Command(fleetctl, fleetArgs...)
	return
}

// Wrapper around fleetctl, able to run every command. It uses two channels to communicate
// output and return code of every command issued.
func FleetExec(args []string, output chan string, exit chan int) {
	var (
		exitCode   int
		waitStatus syscall.WaitStatus
	)
	cmd := FleetPrepareCmd(args)
	if strings.HasPrefix(args[0], "list-") {
		Print(fmt.Sprintf("running %s", strings.Join(cmd.Args, " ")))
	} else {
		PrintD(fmt.Sprintf("running %s", strings.Join(cmd.Args, " ")))
	}

	cmdOut, err := cmd.StdoutPipe()
	if err != nil {
		PrintDE(err)
	}
	cmdErr, err := cmd.StderrPipe()
	if err != nil {
		PrintDE(err)
	}

	scannerOut := bufio.NewScanner(cmdOut)
	scannerErr := bufio.NewScanner(cmdErr)
	go func() {
		for scannerOut.Scan() {
			output <- scannerOut.Text()
		}
		for scannerErr.Scan() {
			output <- scannerErr.Text()
		}
		close(output)
	}()

	if err := cmd.Start(); err != nil {
		if err != nil {
			PrintDE(err)
		}
	}

	if err := cmd.Wait(); err != nil {
		if err != nil {
			PrintDE(err)
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus = exitError.Sys().(syscall.WaitStatus)
			exitCode = waitStatus.ExitStatus()
		}
	}

	PrintD("exit code: " + strconv.Itoa(exitCode))
	exit <- exitCode
	close(exit)
	return
}

// Process output and exit channel from a fleetctl command.
func FleetProcessOutput(output chan string, exit chan int) (exitCode int) {
	for out := range output {
		Print(out)
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
		PrintF(errors.New("invalid unit or maybe you forgot to run maestro build..."))
	}
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
		Print("maestro unit " + unitPath)
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
	exitCode += FleetProcessOutput(output, exit)
	return
}

// Wrapper to run a container build on the coreos cluster.
func FleetBuildUnit(_, unitPath string) (exitCode int) {
	cmds := []string{"destroy", "submit", "load", "start"}
	for _, cmd := range cmds {
		exitCode += FleetExecCommand(cmd, unitPath)
	}
	return
}

// Wrapper to run a unit on the coreos cluster.
func FleetRunUnit(_, unitPath string) (exitCode int) {
	cmds := []string{"submit", "load", "start"}
	if !FleetIsUnitRunning(unitPath) {
		for _, cmd := range cmds {
			exitCode += FleetExecCommand(cmd, unitPath)
		}
	} else {
		Print("unit " + unitPath + " already running")
	}
	return
}
