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
	flagMaestroDir     = app.Flag("maestrodir", "directory on the local host for configs and temporary files (default to $USER/.maestro)").String()
	flagDomain         = app.Flag("domain", "domain used to deal with etcd, skydns, spartito and violino").Default("maestro.io").String()
	flagFleetEndpoints = app.Flag("etcd", "etcd / fleet endpoints to connect").Default("http://172.17.8.101:2379,http://172.17.8.102:2379,http://172.17.8.103:2379").String()
	flagFleetOptions   = app.Flag("fleetopts", "fleetctl options").Strings()

	// cluster
	flagCoreStatus = app.Command("corestatus", "report coreos cluster status")
	flagExec       = app.Command("exec", "exec an arbitrary command through fleet, returning output as stdout and exit code")
	flagExecArgs   = flagExec.Arg("args", "args to fleetclt command").Required().Strings()

	// app
	flagRun         = app.Command("run", "run current app on coreos (this will build unit files, submit and run them)")
	flagRunUnit     = flagRun.Arg("name", "restrict to one component").String()
	flagStop        = app.Command("stop", "stop current app without cleaning unit files on coreos")
	flagStopUnit    = flagStop.Arg("name", "restrict to one component").String()
	flagNuke        = app.Command("nuke", "stop current app and clean unit files on coreos")
	flagNukeAll     = flagNuke.Flag("all", "stop current app and clean ALL unit files on coreos").Bool()
	flagNukeUnit    = flagNuke.Arg("name", "restrict to one component").String()
	flagStatus      = app.Command("status", "show the global app status (systemctl status unitfiles)")
	flagStatusUnit  = flagStatus.Arg("name", "restrict to one component").String()
	flagJournal     = app.Command("journal", "show the journal (journalctl -xu unit) of one app's component")
	flagJournalUnit = flagJournal.Arg("name", "restrict to one component").String()

	// info
	flagUser   = app.Command("user", "get current user name")
	flagConfig = app.Command("config", "print json configuration for current app")

	// build
	flagBuildUnits      = app.Command("build", "locally build app units")
	flagBuildImages     = app.Command("buildimages", "run a container build and registry push on the cluster")
	flagBuildImagesUnit = flagBuildImages.Arg("unit", "restrict to one component").String()
	flagBuildStatus     = app.Command("buildstatus", "check status of a container build and registry push on the cluster")
	flagBuildStatusUnit = flagBuildStatus.Arg("name", "restrict to one component").String()
	flagBuildNuke       = app.Command("buildnuke", "check status of a container build and registry push on the cluster")
	flagBuildNukeUnit   = flagBuildNuke.Arg("name", "restrict to one component").String()
)

func main() {
	var exitCode int
	fleetOutput := make(chan string)
	fleetExitCode := make(chan int)
	args, err := app.Parse(os.Args[1:])
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		os.Exit(1)
	}
	config = maestro.BuildMaestroConfig(*flagConfigFile, *flagMaestroDir, *flagDomain,
		*flagVolumesDir, *flagFleetEndpoints, *flagFleetOptions, *flagDebug)

	// command switch
	exitCode = 0
	switch kingpin.MustParse(args, err) {

	// info
	case flagConfig.FullCommand():
		config.Print()
	case flagUser.FullCommand():
		config.GetUsername()

	// build
	case flagBuildUnits.FullCommand():
		maestro.MaestroBuildLocalUnits()
	case flagBuildImages.FullCommand():
		exitCode = maestro.MaestroBuildContainers(*flagBuildImagesUnit)
	case flagBuildStatus.FullCommand():
		exitCode = maestro.MaestroBuildStatus(*flagBuildStatusUnit)
	case flagBuildNuke.FullCommand():
		exitCode = maestro.MaestroBuildNuke(*flagBuildNukeUnit)

	// exec
	case flagExec.FullCommand():
		go maestro.FleetExec(*flagExecArgs, fleetOutput, fleetExitCode)
		exitCode = maestro.FleetProcessOutput(fleetOutput, fleetExitCode)
	case flagCoreStatus.FullCommand():
		exitCode = maestro.MaestroCoreStatus()
	case flagStatus.FullCommand():
		exitCode = maestro.MaestroStatus(*flagStatusUnit)
	case flagJournal.FullCommand():
		exitCode = maestro.MaestroJournal(*flagJournalUnit)
	case flagRun.FullCommand():
		exitCode = maestro.MaestroRun(*flagRunUnit)
	case flagStop.FullCommand():
		exitCode = maestro.MaestroStop(*flagStopUnit)
	case flagNuke.FullCommand():
		exitCode = maestro.MaestroNuke(*flagNukeUnit)
		if *flagNukeAll {
			exitCode += maestro.MaestroBuildNuke(*flagNukeUnit)
		}
	}

	fmt.Printf("global exit code: %d\n", exitCode)
	os.Exit(exitCode)
}
