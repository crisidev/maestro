package maestro

import "strconv"

// Build local unit files for all components in configuration
func MaestroBuildLocalUnits() {
	for _, stage := range config.Stages {
		for _, component := range stage.Components {
			Print("building component " + username.Username + "/" + config.App + "/" + component.Name)
			ProcessUnitTmpl(component, component.Name, component.UnitPath, "run-unit.tmpl")
		}
	}
}

func MaestroBuildContainers() (exitCode int) {
	for _, stage := range config.Stages {
		for _, component := range stage.Components {
			ProcessUnitTmpl(component, component.Name, component.BuildUnitPath, "build-unit.tmpl")
		}
	}
	return
}

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

func MaestroRun(unit string) (exitCode int) {
	for _, stage := range config.Stages {
		for _, component := range stage.Components {
			for i := 1; i < component.Scale+1; i++ {
				unit = config.GetUnitName(stage.Name, component.Name, strconv.Itoa(i))
				if !IsUnitRunning(unit) {
					unitPath := config.GetNumberedUnitPath(component.UnitPath, strconv.Itoa(i))
					output := make(chan string)
					exit := make(chan int)
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
	return
}

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
