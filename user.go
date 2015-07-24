package maestro

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

var (
	username MaestroUser
	userFile string
)

// MaestroUser structure
type MaestroUser struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

func GetUsername() {
	Print(username.Username)
}

// Wizard to create a new username.
func UsernameWizard() {
	reader := bufio.NewReader(os.Stdin)
	Print("maestro setup wizard")
	PrintR("username: ")
	text, _ := reader.ReadString('\n')
	username.Username = strings.Split(text, "\n")[0]
	PrintR("email: ")
	text, _ = reader.ReadString('\n')
	username.Email = strings.Split(text, "\n")[0]
}

// Writes username JSON informations onto user file
func WriteUsernameFile() {
	PrintD("writing username details")
	data, err := json.Marshal(username)
	if err != nil {
		PrintF(err)
	}
	err = ioutil.WriteFile(userFile, data, 0644)
	if err != nil {
		PrintF(err)
	}
	PrintD("username details saved into " + userFile)
}

// Manages username creation, loading and saving to file.
func SetupUsername() {
	userFile = fmt.Sprintf("%s/%s", maestroDir, "user.json")
	PrintD("user json config file is " + userFile)
	file, err := ioutil.ReadFile(userFile)
	// user config file do not exist. starting wizard.
	if err != nil {
		PrintD("user json config file not found, starting wizard")
		UsernameWizard()
		WriteUsernameFile()
	} else {
		PrintD("user json config file found, loading json")
		// load username details
		err = json.Unmarshal(file, &username)
		if err != nil {
			PrintF(err)
		}
	}
	PrintD("username " + username.Username)
}

// Redo user setup
func SetupUsernameRebuild() {
	PrintD("removing user config files and build directory for " + username.Username)
	err := os.Remove(userFile)
	if err != nil {
		PrintE(err)
	}
	err = os.RemoveAll(path.Join(maestroDir, username.Username))
	if err != nil {
		PrintE(err)
	}
	SetupUsername()
}

// Entrypoint for username module
func GetUserOrRebuild(userChange bool) {
	if userChange {
		PrintD("requested user change")
		SetupUsernameRebuild()
	} else {
		GetUsername()
	}
}
