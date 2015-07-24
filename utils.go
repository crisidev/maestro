package maestro

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

// Print utils
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
