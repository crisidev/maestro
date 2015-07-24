// Maestro: a solution to develop and manage unit files on coreos, gently stuffed with service autodiscovery, load balancing, automatic DNS and a nice build system.
package maestro

import "strconv"

// All functions in this module creates two channels, output and exit, to allow
// fleetctl to write on them and will collect and return output and exit code.

// Build local unit files for all components in configuration.
func MaestroBuildLocalUnits() {
	for _, stage := range config.Stages {
		for _, component := range stage.Components {
			Print("building component " + username.Username + "/" + config.App + "/" + component.Name)
			ProcessUnitTmpl(component, component.Name, component.UnitPath, "run-unit.tmpl")
		}
	}
}

// Build local unit files to build new docker images. After the unit is build, it will
// destroy, submit, load and start using fleetctl. The image will be pushed to the local
// registry.
func MaestroBuildContainers(path string) (exitCode int) {
	output := make(chan string)
	exit := make(chan int)
	if path != "" {
		go FleetExec([]string{"destroy", path}, output, exit)
		exitCode += FleetProcessOutput(output, exit)
		output = make(chan string)
		exit = make(chan int)
		go FleetExec([]string{"submit", path}, output, exit)
		exitCode += FleetProcessOutput(output, exit)
		output = make(chan string)
		exit = make(chan int)
		go FleetExec([]string{"load", path}, output, exit)
		exitCode += FleetProcessOutput(output, exit)
		output = make(chan string)
		exit = make(chan int)
		go FleetExec([]string{"start", path}, output, exit)
		exitCode += FleetProcessOutput(output, exit)
	} else {
		for _, stage := range config.Stages {
			for _, component := range stage.Components {
				ProcessUnitTmpl(component, component.Name, component.BuildUnitPath, "build-unit.tmpl")
				output = make(chan string)
				exit = make(chan int)
				go FleetExec([]string{"destroy", component.BuildUnitPath}, output, exit)
				exitCode += FleetProcessOutput(output, exit)
				output = make(chan string)
				exit = make(chan int)
				go FleetExec([]string{"submit", component.BuildUnitPath}, output, exit)
				exitCode += FleetProcessOutput(output, exit)
				output = make(chan string)
				exit = make(chan int)
				go FleetExec([]string{"load", component.BuildUnitPath}, output, exit)
				exitCode += FleetProcessOutput(output, exit)
				output = make(chan string)
				exit = make(chan int)
				go FleetExec([]string{"start", component.BuildUnitPath}, output, exit)
				exitCode += FleetProcessOutput(output, exit)
			}
		}
	}
	return
}

// Check and prints the status of all units used to build new docker images.
func MaestroBuildStatus(unit string) (exitCode int) {
	output := make(chan string)
	exit := make(chan int)
	if unit != "" {
		go FleetExec([]string{"status", unit}, output, exit)
		exitCode += FleetProcessOutput(output, exit, false)
	} else {
		for _, stage := range config.Stages {
			for _, component := range stage.Components {
				output = make(chan string)
				exit = make(chan int)
				go FleetExec([]string{"status", component.BuildUnitPath}, output, exit)
				exitCode += FleetProcessOutput(output, exit, false)
			}
		}
	}
	return
}

// Utility function to check if a unit is already running on the cluster.
func IsUnitRunning(unit string) (ret bool) {
	ret = false
	output := make(chan string)
	exit := make(chan int)
	go FleetExec([]string{"status", unit}, output, exit)
	_ = <-output
	exitCode := <-exit
	if exitCode == 0 {
		ret = true
	}
	return
}

// Function used to submit, load and start all the units inside the current app.
// It can start also a single unit, using `path` argument. If the unit is already running,
// it will print a message and do nothing.
func MaestroRun(path string) (exitCode int) {
	output := make(chan string)
	exit := make(chan int)
	if path != "" {
		go FleetExec([]string{"submit", path}, output, exit)
		exitCode += FleetProcessOutput(output, exit)
		output = make(chan string)
		exit = make(chan int)
		go FleetExec([]string{"load", path}, output, exit)
		exitCode += FleetProcessOutput(output, exit)
		output = make(chan string)
		exit = make(chan int)
		go FleetExec([]string{"start", path}, output, exit)
		exitCode += FleetProcessOutput(output, exit)
	} else {
		for _, stage := range config.Stages {
			for _, component := range stage.Components {
				for i := 1; i < component.Scale+1; i++ {
					unit := config.GetUnitName(stage.Name, component.Name, strconv.Itoa(i))
					if !IsUnitRunning(unit) {
						unitPath := config.GetNumberedUnitPath(component.UnitPath, strconv.Itoa(i))
						output = make(chan string)
						exit = make(chan int)
						go FleetExec([]string{"submit", unitPath}, output, exit)
						exitCode += FleetProcessOutput(output, exit)
						output = make(chan string)
						exit = make(chan int)
						go FleetExec([]string{"load", unit}, output, exit)
						exitCode += FleetProcessOutput(output, exit)
						output = make(chan string)
						exit = make(chan int)
						go FleetExec([]string{"start", unit}, output, exit)
						exitCode += FleetProcessOutput(output, exit)
					} else {
						Print("Unit " + unit + " already launched")
					}
				}
			}
		}
	}
	return
}

// Stops all units in the current app. It can stop also a single unit, using `unit` argument.
func MaestroStop(unit string) (exitCode int) {
	output := make(chan string)
	exit := make(chan int)
	if unit != "" {
		go FleetExec([]string{"stop", unit}, output, exit)
		exitCode += FleetProcessOutput(output, exit, false)
	} else {
		for _, stage := range config.Stages {
			for _, component := range stage.Components {
				for i := 1; i < component.Scale+1; i++ {
					unit := config.GetUnitName(stage.Name, component.Name, strconv.Itoa(i))
					output := make(chan string)
					exit := make(chan int)
					go FleetExec([]string{"stop", unit}, output, exit)
					exitCode += FleetProcessOutput(output, exit)
				}
			}
		}
	}
	return
}

// Destroys all units in the current app. It can stop also a single unit, using `unit` argument.
func MaestroNuke(unit string) (exitCode int) {
	output := make(chan string)
	exit := make(chan int)
	if unit != "" {
		go FleetExec([]string{"destroy", unit}, output, exit)
		exitCode += FleetProcessOutput(output, exit, false)
	} else {
		for _, stage := range config.Stages {
			for _, component := range stage.Components {
				for i := 1; i < component.Scale+1; i++ {
					unit := config.GetUnitName(stage.Name, component.Name, strconv.Itoa(i))
					output = make(chan string)
					exit = make(chan int)
					go FleetExec([]string{"destroy", unit}, output, exit)
					exitCode += FleetProcessOutput(output, exit)
				}
			}
		}
	}
	return
}

// Prints status for all units in the current app It can also get the status of a single unit, using `unit` argument.
func MaestroStatus(unit string) (exitCode int) {
	output := make(chan string)
	exit := make(chan int)
	if unit != "" {
		go FleetExec([]string{"status", unit}, output, exit)
		exitCode += FleetProcessOutput(output, exit, false)
	} else {
		for _, stage := range config.Stages {
			for _, component := range stage.Components {
				for i := 1; i < component.Scale+1; i++ {
					unit = config.GetUnitName(stage.Name, component.Name, strconv.Itoa(i))
					output = make(chan string)
					exit = make(chan int)
					go FleetExec([]string{"status", unit}, output, exit)
					exitCode += FleetProcessOutput(output, exit, false)
				}
			}
		}
	}
	return
}

// Prints the journal for all units in the current app It can also get the journal of a single unit, using `unit` argument.
func MaestroJournal(unit string) (exitCode int) {
	output := make(chan string)
	exit := make(chan int)
	args := []string{"journal"}
	if unit != "" {
		go FleetExec(append(args, unit), output, exit)
		exitCode += FleetProcessOutput(output, exit, false)
	} else {
		for _, stage := range config.Stages {
			for _, component := range stage.Components {
				for i := 1; i < component.Scale+1; i++ {
					unit = config.GetUnitName(stage.Name, component.Name, strconv.Itoa(i))
					output := make(chan string)
					exit := make(chan int)
					Print("maestro unit: " + unit + "\n")
					go FleetExec(append(args, unit), output, exit)
					exitCode += FleetProcessOutput(output, exit, false)
				}
			}
		}
	}
	return
}

// Executes a global coreos status, running `list-machines`, `list-units`, `list-unit-files`.
func MaestroCoreStatus() (exitCode int) {
	PrintD("executing global status for coreos cluster")
	argsList := [][]string{[]string{"list-machines"}, []string{"list-units"}, []string{"list-unit-files"}}
	for _, args := range argsList {
		output := make(chan string)
		exit := make(chan int)
		go FleetExec(args, output, exit)
		exitCode += FleetProcessOutput(output, exit, false)
	}
	return
}
