// Maestro is a development system able to build and deploy applications (docker images) on a distrubuted CoreOS cluster. Applications are described using a simple JSON configuration and converted into runnable Fleet units and all the burden about namespaces, DNS, container linking, etc is automatically handled by Maestro
//
// Prerequisites
//
// Install Vagrant and Fleetctl and Golang for your architecture.
//
// Installation and Usage
//
// Please read README.md for full installation instructions and usage.
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
			lg.Out("building run unit for " + lg.r(config.Username) + "/" + lg.y(stage.Name) + "/" + lg.g(config.App) + "/" + lg.b(component.Name))
			ProcessUnitTmpl(component, component.Name, component.UnitPath, "run-unit.tmpl")
			if component.BuildUnitPath != "" {
				lg.Out("building build unit for " + lg.r(config.Username) + "/" + lg.y(stage.Name) + "/" + lg.g(config.App) + "/" + lg.b(component.Name))
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
				if cmd == "status" {
					lg.Out(lg.b("maestro ") + "unit: " + unit)
				}
				exitCode += fn(cmd, localPath)
				return
			} else {
				if component.GitSrc != "" {
					if cmd == "status" {
						lg.Out(lg.b("maestro ") + "unit: " + strings.Trim(component.UnitName, "@") + "-build")
					}
					exitCode += fn(cmd, component.BuildUnitPath)
				}
			}
		}
	}
	return
}

// Exec an arbitrary function on a run unit
func MaestroExecRun(fn MaestroCommand, cmd, unit string) (exitCode int) {
	if unit != "" {
		if cmd == "status" {
			lg.Out(lg.b("maestro ") + "unit: " + unit)
		}
		exitCode += fn(cmd, unit)
		return
	} else {
		for _, stage := range config.Stages {
			for _, component := range stage.Components {
				for i := 1; i < component.Scale+1; i++ {
					if cmd == "status" {
						lg.Out(lg.b("maestro ") + "unit: " + component.UnitName + strconv.Itoa(i))
					}
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
	MaestroBuildLocalUnits()
	lg.Out("check results with " + lg.b("maestro buildstatus <unit name>"))
	exitCode = MaestroExecBuild(FleetBuildUnit, "", unit)
	return
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
	exitCode = MaestroExecRun(FleetRunUnit, "", unit)
	lg.Out("check results with " + lg.b("maestro status") + "|" + lg.b("journal <unit name>"))
	return
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
	lg.Out("executing global status for coreos cluster")
	argsList := [][]string{[]string{"list-machines"}, []string{"list-units"}, []string{"list-unit-files"}}
	for i, args := range argsList {
		output := make(chan string)
		exit := make(chan int)
		lg.Out(lg.b("maestro ") + "running fleetctl " + strings.Join(args, " "))
		go FleetExec(args, output, exit)
		exitCode += FleetProcessOutput(output, exit)
		if i < 3 {
			lg.Out("")
		}
	}
	return
}

func MaestroNukeAll() (exitCode int) {
	reader := bufio.NewReader(os.Stdin)
	lg.OutRaw(lg.r("are you sure you want to nuke ALL units on this cluster? [y/N] "))
	text, _ := reader.ReadString('\n')
	if text == "y\n" || text == "Y\n" {
		output := make(chan string)
		exit := make(chan int)
		go FleetExec([]string{"list-units"}, output, exit)
		for line := range output {
			if strings.Contains(line, "service") {
				split := strings.Fields(line)
				localOutput := make(chan string)
				localExit := make(chan int)
				go FleetExec([]string{"destroy", split[0]}, localOutput, localExit)
				exitCode += FleetProcessOutput(localOutput, localExit)
			}
		}
		_ = <-exit
		output = make(chan string)
		exit = make(chan int)
		go FleetExec([]string{"list-unit-files"}, output, exit)
		for line := range output {
			if strings.Contains(line, "service") {
				split := strings.Fields(line)
				localOutput := make(chan string)
				localExit := make(chan int)
				go FleetExec([]string{"destroy", split[0]}, localOutput, localExit)
				exitCode += FleetProcessOutput(localOutput, localExit)
			}
		}
		_ = <-exit
	}
	return
}

func MaestroCommandExec(cmd *exec.Cmd, output chan string) (exitCode int) {
	cmdOut, err := cmd.StdoutPipe()
	if err != nil {
		lg.DebugError(err)
	}
	cmdErr, err := cmd.StderrPipe()
	if err != nil {
		lg.DebugError(err)
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
		lg.DebugError(err)
	}
	if err := cmd.Wait(); err != nil {
		if err != nil {
			lg.DebugError(err)
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			exitCode = waitStatus.ExitStatus()
		}
	}
	return
}

func HandleExit(exitCode int) {
	if exitCode > 0 {
		lg.Out(lg.b("maestro ") + "exit code: " + lg.r(strconv.Itoa(exitCode)))
	} else {
		lg.Out(lg.b("maestro ") + "exit code: " + lg.g(strconv.Itoa(exitCode)))
	}
	os.Exit(exitCode)
}
