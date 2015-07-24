package main

import (
	"fmt"
	"os"

	"github.com/crisidev/maestro"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	APP_NAME    = "maestro"
	APP_VERSION = "0.0.1"
	APP_AUTHOR  = "bigo@crisidev.org"
	APP_SITE    = "https://github.com/crisidev/maestro"
)

var (
	config maestro.MaestroConfig
	app    = kingpin.New(APP_NAME, fmt.Sprintf("friendly generates and deploy systemd unit files on a CoreOS maestro cluster %s", APP_SITE))

	// global
	flagDebug          = app.Flag("debug", "enable debug mode.").Bool()
	flagConfigFile     = app.Flag("config", "configuration file").Default("maestro.json").String()
	flagVolumesDir     = app.Flag("volumesdir", "directory on the coreos host for shared volumes").Default("/var/maestro").String()
	flagMaestroDir     = app.Flag("buildir", "directory on the local host for temporary storing of information").Default(".maestro").String()
	flagDomain         = app.Flag("domain", "domain used to deal with etcd, skydns, spartito and violino").Default("maestro.io").String()
	flagFleetEndpoints = app.Flag("etcd", "etcd / fleet endpoints to connect").Default("http://172.17.8.101:2379,http://172.17.8.102:2379,http://172.17.8.103:2379").String()
	flagFleetOptions   = app.Flag("fleetopts", "fleetctl options").Strings()

	// cluster
	flagCoreStatus = app.Command("corestatus", "report coreos cluster status")
	flagExec       = app.Command("exec", "exec an arbitrary command through fleet, returning output as stdout and exit code")
	flagExecArgs   = flagExec.Arg("args", "args to fleetclt command").Required().Strings()

	// app
	flagAppRun         = app.Command("run", "run current app on coreos (this will build unit files, submit and run them)")
	flagAppRunUnit     = flagAppRun.Arg("name", "restrict to one component").String()
	flagAppStop        = app.Command("stop", "stop current app without cleaning unit files on coreos")
	flagAppStopUnit    = flagAppStop.Arg("name", "restrict to one component").String()
	flagAppNuke        = app.Command("nuke", "stop current app and clean unit files on coreos")
	flagAppNukeUnit    = flagAppNuke.Arg("name", "restrict to one component").String()
	flagAppStatus      = app.Command("status", "show the global app status (systemctl status unitfiles)")
	flagAppStatusUnit  = flagAppStatus.Arg("name", "restrict to one component").String()
	flagAppJournal     = app.Command("journal", "show the journal (journalctl -xu unit) of one app's component")
	flagAppJournalUnit = flagAppJournal.Arg("name", "restrict to one component").String()

	// info
	flagUser       = app.Command("user", "get current user name")
	flagUserChange = flagUser.Flag("changeuser", "remove current user config and start the wizard again").Bool()
	flagConfig     = app.Command("config", "print json configuration for current app")

	// build
	flagBuildUnits          = app.Command("build", "locally build app units")
	flagBuildContainers     = app.Command("buildcontainers", "run a container build and registry push on the cluster")
	flagBuildContainersUnit = flagBuildContainers.Command("name", "restrict to one component")
)

func main() {
	var exitCode int
	fleetOutput := make(chan string)
	fleetExitCode := make(chan int)
	args, err := app.Parse(os.Args[1:])
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		os.Exit(1)
	}
	maestro.SetupUsername()
	config = maestro.ParseMaestroConfig(*flagConfigFile, *flagDomain, *flagVolumesDir, *flagFleetEndpoints, *flagFleetOptions, *flagDebug)
	maestro.SetupMaestroAppDirs()

	// command switch
	exitCode = 0
	switch kingpin.MustParse(args, err) {

	// info
	case flagConfig.FullCommand():
		config.Print()
	case flagUser.FullCommand():
		maestro.GetUserOrRebuild(*flagUserChange)

	// build
	case flagBuildUnits.FullCommand():
		maestro.MaestroBuildLocalUnits()
	case flagBuildContainers.FullCommand():
		exitCode = maestro.MaestroBuildContainers()

	// exec
	case flagExec.FullCommand():
		go maestro.FleetExec(*flagExecArgs, fleetOutput, fleetExitCode)
		exitCode = maestro.FleetProcessOutput(fleetOutput, fleetExitCode)
	case flagCoreStatus.FullCommand():
		exitCode = maestro.MaestroCoreStatus()
	case flagAppStatus.FullCommand():
		exitCode = maestro.MaestroStatus(*flagAppStatusUnit)
	case flagAppJournal.FullCommand():
		exitCode = maestro.MaestroJournal(*flagAppJournalUnit)
	case flagAppRun.FullCommand():
		exitCode = maestro.MaestroRun(*flagAppRunUnit)
	case flagAppStop.FullCommand():
		exitCode = maestro.MaestroStop(*flagAppStopUnit)
	case flagAppNuke.FullCommand():
		exitCode = maestro.MaestroNuke(*flagAppNukeUnit)
	}

	fmt.Printf("global exit code: %d\n", exitCode)
	os.Exit(exitCode)
}
