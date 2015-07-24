package maestro

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
)

var maestroDir string

// setup directory (/home/$USER/.maestro) used to store user informations and
// temporary build file before submitting to coreos. directory will be created if not existent.
func SetupMaestroDir() {
	cwd, err := os.Getwd()
	if err != nil {
		PrintF(err)
	}
	maestroDir = fmt.Sprintf("%s/%s", cwd, ".maestro")
	if _, err := os.Stat(maestroDir); err != nil {
		PrintD(maestroDir + " not found, creating")
		os.Mkdir(maestroDir, 0755)
	}
}

// created directories for unit file buildint
// schema: /home/$USER/.maestro/$username/$stage/$app/$component/
func SetupMaestroAppDirs() {
	PrintD("creating build dirs for app and components")
	os.Mkdir(fmt.Sprintf("%s/%s", maestroDir, config.Username), 0755)
	for _, stage := range config.Stages {
		os.Mkdir(path.Join(maestroDir, config.Username, stage.Name), 0755)
		os.Mkdir(path.Join(maestroDir, config.Username, stage.Name, config.App), 0755)
	}
}

// print utils
func PrintE(err error) {
	if err != nil {
		Print(fmt.Sprintf("error: %s", err.Error()))
	}
}

func PrintF(err error) {
	if err != nil {
		Print(fmt.Sprintf("error: %s", err.Error()))
		os.Exit(1)
	}
}
func PrintU() {
	PrintE(errors.New("command not implemented yet ;)"))
}

func PrintW(out string) {
	log.Println("warn: " + out)
}

func PrintD(out string) {
	if flagDebug {
		if config.App == "" {
			log.Println(fmt.Sprintf("debug: " + out))
		} else {
			log.Println(fmt.Sprintf("debug: A %s: %s", config.App, out))
		}
	}
}

func PrintDC(name, out string) {
	if flagDebug {
		log.Println(fmt.Sprintf("debug: C %s/%s: %s", config.App, strings.Replace(name, "@", "", 1), out))
	}
}

func PrintR(out string) {
	if len(out) > 0 {
		fmt.Printf("%s", out)
	}
}

func Print(out string) {
	fmt.Println(out)
}
