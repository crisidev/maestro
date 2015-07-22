package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// fleetctl execs
func FleetExec(args []string, output chan string, exit chan int) {
	var (
		waitStatus syscall.WaitStatus
		cmdOut     bytes.Buffer
		cmdErr     bytes.Buffer
		out        string
	)

	exitCode := -1
	//cmdOut := &bytes.Buffer{}
	//cmdErr := &bytes.Buffer{}
	fleetArgs := []string{"--endpoint", *flagFleetEndpoints}
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
		out = string(cmdOut.Bytes())
	}
	PrintD("output: " + out)
	PrintD("exit code: " + strconv.Itoa(exitCode))
	output <- out
	exit <- exitCode
	close(output)
	close(exit)
	return
}

func FleetExecCommand(args []string, output chan string, exit chan int) {
	go FleetExec(args, output, exit)
}

func FleetExecStatus(unit string, output chan string, exit chan int) {
	args := []string{"status", unit}
	go FleetExec(args, output, exit)
}

func FleetExecJournal(unit string, output chan string, exit chan int) {
	args := []string{"journal", unit}
	go FleetExec(args, output, exit)
}

func FleetExecSubmit(unitPath string, output chan string, exit chan int) {
	args := []string{"submit", unitPath}
	go FleetExec(args, output, exit)
}

func FleetExecLoad(unitPath string, output chan string, exit chan int) {
	args := []string{"load", unitPath}
	go FleetExec(args, output, exit)
}

func FleetExecUnload(unitPath string, output chan string, exit chan int) {
	args := []string{"unload", unitPath}
	go FleetExec(args, output, exit)
}

func FleetExecStart(unitPath string, output chan string, exit chan int) {
	args := []string{"start", unitPath}
	go FleetExec(args, output, exit)
}

func FleetExecStop(unitPath string, output chan string, exit chan int) {
	args := []string{"stop", unitPath}
	go FleetExec(args, output, exit)
}

func FleetExecDestroy(unitPath string, output chan string, exit chan int) {
	args := []string{"destroy", unitPath}
	go FleetExec(args, output, exit)
}

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
