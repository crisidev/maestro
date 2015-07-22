package main

import (
	"fmt"
	"os"

	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	APP_NAME    = "maestro"
	APP_VERSION = "0.0.1"
	APP_AUTHOR  = "bigo@crisidev.org"
	APP_SITE    = "https://github.com/crisidev/maestro"
)

var (
	// vars
	username   MaestroUser
	config     MaestroConfig
	userFile   string
	maestroDir string
	app        = kingpin.New(APP_NAME, fmt.Sprintf("friendly generates and deploy systemd unit files on a CoreOS maestro cluster %s", APP_SITE))

	// global
	flagDebug          = app.Flag("debug", "enable debug mode.").Bool()
	flagConfigFile     = app.Flag("config", "configuration file").Default("maestro.json").String()
	flagVolumesDir     = app.Flag("volumesdir", "directory on the coreos host for shared volumes").Default("/var/maestro").String()
	flagRuntimeDir     = app.Flag("runtimedir", "directory on the coreos host for runtime files").Default("/run/maestro").String()
	flagMaestroDir     = app.Flag("buildir", "directory on the local host for temporary storing of information").Default(".maestro").String()
	flagDomain         = app.Flag("domain", "domain used to deal with etcd, skydns, spartito and violino").Default("maestro.io").String()
	flagFleetEndpoints = app.Flag("etcd", "etcd / fleet endpoints to connect").Default("http://172.17.8.101:2379,http://172.17.8.102:2379,http://172.17.8.103:2379").String()

	// cluster
	flagCoreStatus = app.Command("corestatus", "report coreos cluster status")
	flagExec       = app.Command("exec", "exec an arbitrary command through fleet, returning output as stdout and exit code")
	flagExecArgs   = flagExec.Arg("args", "args to fleetclt command").Required().Strings()

	// app
	flagAppRun         = app.Command("run", "run current app on coreos (this will build unit files, submit and run them)")
	flagAppStop        = app.Command("stop", "stop current app without cleaning unit files on coreos")
	flagAppNuke        = app.Command("nuke", "stop current app and clean unit files on coreos")
	flagAppStatus      = app.Command("status", "show the global app status (systemctl status unitfiles)")
	flagAppStatusUnit  = flagAppStatus.Arg("name", "restrict to one component").String()
	flagAppJournal     = app.Command("journal", "show the journal (journalctl -xu unit) of one app's component")
	flagAppJournalUnit = flagAppJournal.Arg("name", "reatrict to one component").String()

	// info
	flagUser       = app.Command("user", "get current user name")
	flagUserChange = flagUser.Flag("changeuser", "remove current user config and start the wizard again").Bool()
	flagConfig     = app.Command("config", "print json configuration for current app")

	// build
	flagBuildLocalUnits = app.Command("buildlocalunits", "locally build app units")
	flagBuildContainers = app.Command("buildcontainers", "run a container build and registry push on the cluster")
)

func MaestroBuildLocalUnits() {
	for _, stage := range config.Stages {
		for _, component := range stage.Components {
			ProcessUnitTmpl(component, component.Name, component.UnitPath, "run-unit.tmpl")
		}
	}
}

func MaestroBuildContainers() (exitCode int) {
	for _, stage := range config.Stages {
		for _, component := range stage.Components {
			ProcessUnitTmpl(component, component.Name, component.BuildUnitPath, "build-unit.tmpl")
			//output := make(chan string)
			//exit := make(chan int)
			//go FleetExecSubmit(component.BuildUnitPath, output, exit)
			//exitCode += FleetProcessOutput(output, exit)
			//output = make(chan string)
			//exit = make(chan int)
			//go FleetExecLoad(component.BuildUnitPath, output, exit)
			//exitCode += FleetProcessOutput(output, exit)
			//output = make(chan string)
			//exit = make(chan int)
			//go FleetExecStart(component.BuildUnitPath, output, exit)
			//exitCode += FleetProcessOutput(output, exit)
		}
	}
	return
}

func MaestroRun() (exitCode int) {
	MaestroBuildLocalUnits()
	for _, stage := range config.Stages {
		for _, component := range stage.Components {
			for i := 1; i < component.Scale+1; i++ {
				unitPath := GetNumberedUnitPath(component.UnitPath, i)
				output := make(chan string)
				exit := make(chan int)
				go FleetExecSubmit(unitPath, output, exit)
				exitCode += FleetProcessOutput(output, exit)
				output = make(chan string)
				exit = make(chan int)
				go FleetExecLoad(unitPath, output, exit)
				exitCode += FleetProcessOutput(output, exit)
				output = make(chan string)
				exit = make(chan int)
				go FleetExecStart(unitPath, output, exit)
				exitCode += FleetProcessOutput(output, exit)
			}
		}
	}
	return
}

func MaestroStop() (exitCode int) {
	for _, stage := range config.Stages {
		for _, component := range stage.Components {
			for i := 1; i < component.Scale+1; i++ {
				unitPath := GetNumberedUnitPath(component.UnitPath, i)
				output := make(chan string)
				exit := make(chan int)
				go FleetExecStop(unitPath, output, exit)
				exitCode += FleetProcessOutput(output, exit)
			}
		}
	}
	return
}

func MaestroNuke() (exitCode int) {
	for _, stage := range config.Stages {
		for _, component := range stage.Components {
			for i := 1; i < component.Scale+1; i++ {
				unitPath := GetNumberedUnitPath(component.UnitPath, i)
				output := make(chan string)
				exit := make(chan int)
				go FleetExecStop(unitPath, output, exit)
				exitCode += FleetProcessOutput(output, exit)
				output = make(chan string)
				exit = make(chan int)
				go FleetExecUnload(unitPath, output, exit)
				output = make(chan string)
				exit = make(chan int)
				go FleetExecDestroy(unitPath, output, exit)
				exitCode += FleetProcessOutput(output, exit)
			}
		}
	}
	return
}

func MaestroStatus() (exitCode int) {
	if *flagAppJournalUnit != "" {
		output := make(chan string)
		exit := make(chan int)
		go FleetExecStatus(*flagAppJournalUnit, output, exit)
	} else {
		for _, stage := range config.Stages {
			for _, component := range stage.Components {
				for i := 1; i < component.Scale+1; i++ {
					unitPath := GetNumberedUnitPath(component.UnitPath, i)
					output := make(chan string)
					exit := make(chan int)
					go FleetExecStatus(unitPath, output, exit)
					exitCode += FleetProcessOutput(output, exit, false)
				}
			}
		}
	}
	return
}

func MaestroJournal() (exitCode int) {
	if *flagAppJournalUnit != "" {
		output := make(chan string)
		exit := make(chan int)
		go FleetExecJournal(*flagAppJournalUnit, output, exit)
		exitCode += FleetProcessOutput(output, exit, false)
	} else {
		for _, stage := range config.Stages {
			for _, component := range stage.Components {
				for i := 1; i < component.Scale+1; i++ {
					unitPath := GetNumberedUnitPath(component.UnitPath, i)
					output := make(chan string)
					exit := make(chan int)
					go FleetExecJournal(unitPath, output, exit)
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

func main() {
	var exitCode int
	fleetOutput := make(chan string)
	fleetExitCode := make(chan int)
	args, err := app.Parse(os.Args[1:])
	if err != nil {
		PrintF(err)
	}
	SetupMaestroDir()
	SetupUsername()
	ParseMaestroConfig()
	SetupMaestroAppDirs()

	// command switch
	exitCode = 0
	switch kingpin.MustParse(args, err) {

	// info
	case flagConfig.FullCommand():
		config.Print()
	case flagUser.FullCommand():
		GetUserOrRebuild()

	// build
	case flagBuildLocalUnits.FullCommand():
		MaestroBuildLocalUnits()
	case flagBuildContainers.FullCommand():
		exitCode = MaestroBuildContainers()

	// exec
	case flagExec.FullCommand():
		go FleetExecCommand(*flagExecArgs, fleetOutput, fleetExitCode)
		exitCode = FleetProcessOutput(fleetOutput, fleetExitCode)
	case flagCoreStatus.FullCommand():
		exitCode = MaestroCoreStatus()
	case flagAppStatus.FullCommand():
		exitCode = MaestroStatus()
	case flagAppJournal.FullCommand():
		exitCode = MaestroJournal()
	case flagAppRun.FullCommand():
		exitCode = MaestroRun()
	case flagAppStop.FullCommand():
		exitCode = MaestroStop()
	case flagAppNuke.FullCommand():
		exitCode = MaestroNuke()
	}

	Print(fmt.Sprintf("global exit code: %d", exitCode))
	os.Exit(exitCode)
}
