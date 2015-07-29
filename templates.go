package maestro

import (
	"os"
	"strings"
	"text/template"
)

// Return a string containing a template.
func GetTmpl(name string) string {
	data, err := Asset("templates/" + name)
	lg.Fatal(err)
	return string(data)
}

// Renders a template onto a file.
func ProcessUnitTmpl(component MaestroComponent, unitName, unitPath, tmplName string) {
	// Little function to cut the domain from the dns
	funcMap := template.FuncMap{
		"cutDomain": func(s string) string {
			return strings.Replace(s, ".maestro.io", "", 1)
		},
	}

	fd := GetUnitFd(unitPath)
	lg.Debug("getting template " + tmplName + " from asset data")
	tmpl, err := template.New(unitName).Funcs(funcMap).Parse(GetTmpl(tmplName))
	lg.Fatal(err)
	lg.Debug("processing template into " + unitPath)
	err = tmpl.Execute(fd, component)
	lg.Fatal(err)
	fd.Close()
}

// Return a file descriptor to be used to render a template.
func GetUnitFd(filepath string) *os.File {
	fd, err := os.Create(filepath)
	lg.Fatal(err)
	return fd
}
