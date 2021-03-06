package main

import (
	"fmt"
	"os"

	"github.com/crisidev/maestro"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	APP_NAME    = "maestro"
	APP_VERSION = "0.0.3"
	APP_AUTHOR  = "bigo@crisidev.org"
	APP_SITE    = "https://github.com/crisidev/maestro"
)

var (
	config maestro.MaestroConfig
	app    = kingpin.New(APP_NAME, fmt.Sprintf("friendly generates and deploy systemd unit files on a CoreOS maestro cluster %s", APP_SITE))

	// global
	flagDebug          = app.Flag("debug", "enable debug mode").Short('d').Bool()
	flagConfigFile     = app.Flag("config", "configuration file").Short('c').Default("maestro.json").String()
	flagVolumesDir     = app.Flag("volumesdir", "directory on the coreos host for shared volumes").Short('V').Default("/share/maestro").String()
	flagMaestroDir     = app.Flag("maestrodir", "directory on the local host for configs and temporary files (default to $USER/.maestro)").Short('m').String()
	flagDomain         = app.Flag("domain", "domain used to deal with etcd, skydns, spartito and violino").Default("maestro.io").String()
	flagFleetEndpoints = app.Flag("etcd", "etcd / fleet endpoints to connect").Short('e').String()
	flagFleetOptions   = app.Flag("fleetopts", "fleetctl options").Short('F').Strings()
	flagFleetAddress   = app.Flag("fleetaddr", "fleetctl tunnel address and port").Default("172.17.8.101").Short('A').String()

	// cluster
	flagCoreStatus = app.Command("corestatus", "report coreos cluster status")
	flagExec       = app.Command("exec", "exec an arbitrary command through fleet, returning output as stdout and exit code")
	flagExecArgs   = flagExec.Arg("args", "args to fleetclt command").Required().Strings()
	flagEtcd       = app.Command("etcd", "get maestro related list of keys from etcd")
	flagEtcdKey    = flagEtcd.Arg("name", "get one key").String()
	flagEtcdSkydns = flagEtcd.Flag("skydns", "include skydns in the list of etcd keys").Short('D').Bool()
	flagEtcdAll    = flagEtcd.Flag("all", "get the list of all etcd keys").Short('a').Bool()

	// app
	flagRun           = app.Command("run", "run current app on coreos (this will build unit files, submit and run them)")
	flagRunUnit       = flagRun.Arg("name", "restrict to one component").String()
	flagStop          = app.Command("stop", "stop current app without cleaning unit files on coreos")
	flagStopUnit      = flagStop.Arg("name", "restrict to one component").String()
	flagNuke          = app.Command("nuke", "stop current app and clean unit files on coreos")
	flagNukeAll       = flagNuke.Flag("all", "stop current app and clean ALL unit files on coreos").Short('a').Bool()
	flagNukeUnit      = flagNuke.Arg("name", "restrict to one component").String()
	flagStatus        = app.Command("status", "show the global app status (systemctl status unitfiles)")
	flagStatusUnit    = flagStatus.Arg("name", "restrict to one component").String()
	flagJournal       = app.Command("journal", "show the journal (journalctl -xu unit) of one app's component")
	flagJournalUnit   = flagJournal.Arg("name", "restrict to one component").String()
	flagJournalFollow = flagJournal.Flag("follow", "follow component journal").Short('f').Bool()
	flagJournalAll    = flagJournal.Flag("all", "get all component journal").Bool()

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

// Command switch for commands requiring a config to be loaded
func ConfigCommandSwitch(args string, err error) (exitCode int) {
	config = maestro.BuildMaestroConfig(*flagConfigFile)
	switch kingpin.MustParse(args, err) {
	case flagConfig.FullCommand():
		config.Print()
	case flagUser.FullCommand():
		config.GetUsername()
	case flagBuildUnits.FullCommand():
		maestro.MaestroBuildLocalUnits()
	case flagBuildImages.FullCommand():
		exitCode = maestro.MaestroBuildContainers(*flagBuildImagesUnit)
	case flagBuildStatus.FullCommand():
		exitCode = maestro.MaestroBuildStatus(*flagBuildStatusUnit)
	case flagBuildNuke.FullCommand():
		exitCode = maestro.MaestroBuildNuke(*flagBuildNukeUnit)
	case flagStatus.FullCommand():
		exitCode = maestro.MaestroStatus("")
	case flagJournal.FullCommand():
		exitCode = maestro.MaestroJournal("", *flagJournalFollow, *flagJournalAll)
	case flagRun.FullCommand():
		exitCode = maestro.MaestroRun(*flagRunUnit)
	case flagStop.FullCommand():
		exitCode = maestro.MaestroStop("")
	case flagNuke.FullCommand():
		exitCode = maestro.MaestroNuke("")
	}
	return
}

// Initial switch for commands not requiring a configuration
func NoConfigCommandSwitch(args string, err error) (exitCode int) {
	exitCode = -1
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case flagCoreStatus.FullCommand():
		exitCode = maestro.MaestroCoreStatus()
	case flagExec.FullCommand():
		output := make(chan string)
		exit := make(chan int)
		go maestro.FleetExec(*flagExecArgs, output, exit)
		exitCode = maestro.FleetProcessOutput(output, exit)
	case flagEtcd.FullCommand():
		exitCode = maestro.EtcdPullKeys(*flagEtcdSkydns, *flagEtcdAll, *flagEtcdKey)
	case flagNuke.FullCommand():
		if *flagNukeAll {
			exitCode = maestro.MaestroNukeAll()
		} else if *flagNukeUnit != "" {
			exitCode = maestro.MaestroNuke(*flagNukeUnit)
		}
	case flagStatus.FullCommand():
		if *flagStatusUnit != "" {
			exitCode = maestro.MaestroStatus(*flagStatusUnit)
		}
	case flagJournal.FullCommand():
		if *flagJournalUnit != "" {
			exitCode = maestro.MaestroJournal(*flagJournalUnit, *flagJournalFollow, *flagJournalAll)
		}
	case flagStop.FullCommand():
		if *flagStopUnit != "" {
			exitCode = maestro.MaestroStop(*flagStopUnit)
		}
	}
	return
}

func main() {
	args, err := app.Parse(os.Args[1:])
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		os.Exit(1)
	}
	// initialize maestro
	maestro.Init(*flagMaestroDir, *flagDomain, *flagFleetAddress,
		*flagVolumesDir, *flagFleetEndpoints, *flagFleetOptions, *flagDebug)

	exitCode := NoConfigCommandSwitch(args, err)
	if exitCode != -1 {
		maestro.HandleExit(exitCode)
	}

	maestro.HandleExit(ConfigCommandSwitch(args, err))
}
