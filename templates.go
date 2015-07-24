package maestro

import (
	"html/template"
	"os"
)

func GetTmpl(name string) string {
	data, err := Asset("files/templates/" + name)
	if err != nil {
		PrintE(err)
	}
	return string(data)
}

func ProcessUnitTmpl(component MaestroComponent, unitName, unitPath, tmplName string) {
	fd := GetUnitFd(unitPath)
	PrintD("getting template " + tmplName + " from asset data")
	tmpl, err := template.New(unitName).Parse(GetTmpl(tmplName))
	if err != nil {
		PrintF(err)
	}
	PrintD("processing template into " + unitPath)
	err = tmpl.Execute(fd, component)
	if err != nil {
		PrintF(err)
	}
	fd.Close()
}

func GetUnitFd(filepath string) *os.File {
	fd, err := os.Create(filepath)
	if err != nil {
		PrintE(err)
	}
	return fd
}
