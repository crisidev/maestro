// Maestro: a solution to develop and manage unit files on coreos, gently stuffed with service autodiscovery, load balancing, automatic DNS and a nice build system.
package maestro

import (
	"bufio"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
)

// Build local unit files for all components in configuration.
func MaestroBuildLocalUnits() {
	for _, stage := range config.Stages {
		for _, component := range stage.Components {
			Print("building component " + config.Username + "/" + config.App + "/" + component.Name)
			ProcessUnitTmpl(component, component.Name, component.UnitPath, "run-unit.tmpl")
			if component.BuildUnitPath != "" {
				Print("building component docker image builder for " + config.Username + "/" + config.App + "/" + component.Name)
				ProcessUnitTmpl(component, component.Name, component.BuildUnitPath, "build-unit.tmpl")
			}
		}
	}
}

// Function passed to commands
type MaestroCommand func(string, string) int

// Exec an arbitrary function on a build unit
func MaestroExecBuild(fn MaestroCommand, cmd, unit string) (exitCode int) {
	for _, stage := range config.Stages {
		for _, component := range stage.Components {
			if unit != "" {
				localPath := path.Join(config.GetAppPath(stage.Name), unit)
				exitCode += fn(cmd, localPath)
				return
			} else {
				if component.GitSrc != "" {
					exitCode += fn(cmd, component.BuildUnitPath)
				}
			}
		}
	}
	return
}

// Exec an arbitrary function on a run unit
func MaestroExecRun(fn MaestroCommand, cmd, unit string) (exitCode int) {
	for _, stage := range config.Stages {
		for _, component := range stage.Components {
			for i := 1; i < component.Scale+1; i++ {
				if unit != "" {
					localPath := path.Join(config.GetAppPath(stage.Name), unit)
					exitCode += fn(cmd, localPath)
					return
				} else {
					exitCode += fn(cmd, config.GetNumberedUnitPath(component.UnitPath, strconv.Itoa(i)))
				}
			}
		}
	}
	return
}

// Build local unit files to build new docker images. After the unit is build, it will
// destroy, submit, load and start using fleetctl. The image will be pushed to the local
// registry.
func MaestroBuildContainers(unit string) (exitCode int) {
	return MaestroExecBuild(FleetBuildUnit, "", unit)
}

// Check and prints the status of all units used to build new docker images.
func MaestroBuildStatus(unit string) (exitCode int) {
	return MaestroExecBuild(FleetExecCommand, "status", unit)
}

// Destroys all units used for building docker images. It can stop also a single unit, using `unit` argument.
func MaestroBuildNuke(unit string) (exitCode int) {
	return MaestroExecBuild(FleetExecCommand, "destroy", unit)
}

// Function used to submit, load and start all the units inside the current app.
// It can start also a single unit, using `unit` argument. If the unit is already running,
// it will print a message and do nothing.
func MaestroRun(unit string) (exitCode int) {
	MaestroBuildLocalUnits()
	return MaestroExecRun(FleetRunUnit, "", unit)
}

// Stops all units in the current app. It can stop also a single unit, using `unit` argument.
func MaestroStop(unit string) (exitCode int) {
	return MaestroExecRun(FleetExecCommand, "stop", unit)
}

// Destroys all units in the current app. It can stop also a single unit, using `unit` argument.
func MaestroNuke(unit string) (exitCode int) {
	return MaestroExecRun(FleetExecCommand, "destroy", unit)
}

// Prints status for all units in the current app It can also get the status of a single unit, using `unit` argument.
func MaestroStatus(unit string) (exitCode int) {
	return MaestroExecRun(FleetExecCommand, "status", unit)
}

// Prints the journal for all units in the current app It can also get the journal of a single unit, using `unit` argument.
func MaestroJournal(unit string, follow, all bool) (exitCode int) {
	cmd := "journal"
	if follow {
		cmd = "journalf"
	} else if all {
		cmd = "journala"
	}
	return MaestroExecRun(FleetExecCommand, cmd, unit)
}

// Executes a global coreos status, running `list-machines`, `list-units`, `list-unit-files`.
func MaestroCoreStatus() (exitCode int) {
	PrintD("executing global status for coreos cluster")
	argsList := [][]string{[]string{"list-machines"}, []string{"list-units"}, []string{"list-unit-files"}}
	for _, args := range argsList {
		output := make(chan string)
		exit := make(chan int)
		go FleetExec(args, output, exit)
		exitCode += FleetProcessOutput(output, exit)
	}
	return
}

func MaestroNukeAll() (exitCode int) {
	reader := bufio.NewReader(os.Stdin)
	output := make(chan string)
	exit := make(chan int)
	go FleetExec([]string{"list-unit-files"}, output, exit)
	PrintR("are you sure you want to nuke ALL units on this cluster? [y/N] ")
	text, _ := reader.ReadString('\n')
	if text == "y\n" || text == "Y\n" {
		for line := range output {
			line = string(line)
			if strings.Contains(line, "service") {
				split := strings.Fields(line)
				localOutput := make(chan string)
				localExit := make(chan int)
				go FleetExec([]string{"destroy", split[0]}, localOutput, localExit)
				exitCode += FleetProcessOutput(localOutput, localExit)
			}
		}
	}
	exitCode = <-exit
	return
}

func MaestroExec(cmd *exec.Cmd, output chan string) (exitCode int) {
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
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			exitCode = waitStatus.ExitStatus()
		}
	}
	return
}
