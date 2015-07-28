package maestro

import (
	"fmt"
	"log"
	"os"

	"github.com/fatih/color"
)

// Global logger
var lg MaestroLog

// Logger with function to print colors
type MaestroLog struct {
	debug  *log.Logger
	output *log.Logger
	br     colorFn
	y      colorFn
	r      colorFn
	b      colorFn
	w      colorFn
	c      colorFn
	g      colorFn
	m      colorFn
	base   string
}

// Damn faith/color which is not exposing a color function
type colorFn func(a ...interface{}) string

func init() {
	lg.output = log.New(os.Stdout, "", 0)
	lg.y = color.New(color.FgYellow).Add(color.Faint).SprintFunc()
	lg.r = color.New(color.FgRed).Add(color.Faint).SprintFunc()
	lg.b = color.New(color.FgBlue).Add(color.Faint).SprintFunc()
	lg.w = color.New(color.FgWhite).Add(color.Faint).SprintFunc()
	lg.c = color.New(color.FgCyan).Add(color.Faint).SprintFunc()
	lg.g = color.New(color.FgGreen).Add(color.Faint).SprintFunc()
	lg.m = color.New(color.FgMagenta).Add(color.Faint).SprintFunc()
	lg.br = color.New(color.FgRed).Add(color.Bold).SprintFunc()
	lg.base = fmt.Sprintf("/%s", lg.g("maestro"))
}

// Setup "base" application name for debugging
func (l MaestroLog) SetupBase() {
	lg.base = fmt.Sprintf("/%s", lg.g(config.App))
}

// Setup debugging to stderr
func (l MaestroLog) SetupDebug() {
	lg.debug = log.New(os.Stderr, "", 0)
	lg.debug.SetFlags(log.Ldate | log.Ltime)
}

// Setup prefix for debugging with colors
func (l MaestroLog) SetupPrefix(prefix ...string) (base string) {
	base = lg.base
	fn := ""
	if len(prefix) > 0 {
		for i, p := range prefix {
			switch i {
			case 0:
				fn = l.y(p)
			case 1:
				fn = l.b(p)
			default:
				fn = l.w(p)
			}
			base = fmt.Sprintf("%s/%s", base, fn)
		}
	}
	return
}

// Logs to stardard output
func (l MaestroLog) Out(msg string) {
	l.output.Println(msg)
}

// Logs to standard output without carriage return
func (l MaestroLog) OutRaw(msg string) {
	fmt.Printf(msg)
}

// Logs to standard error with colors
func (l MaestroLog) Debug(msg string, prefix ...string) {
	base := l.SetupPrefix(prefix...)
	if lg.debug != nil {
		l.debug.Printf("%s: %s", base, msg)
	}
}

// Logs to standard error with colors also in the message
func (l MaestroLog) Debug2(msg, suffix string, prefix ...string) {
	base := l.SetupPrefix(prefix...)
	if l.debug != nil {
		l.debug.Printf("%s: %s %s", base, msg, l.c(suffix))
	}
}

// Debugs an error
func (l MaestroLog) DebugError(err error) {
	base := l.SetupPrefix()
	if err != nil && l.debug != nil {
		msg := fmt.Sprintf(lg.r("error") + ": " + err.Error())
		l.debug.Printf("%s: %s", base, msg)
	}
}

// Logs an error
func (l MaestroLog) Error(err error) {
	if err != nil {
		l.output.Printf("%s: %s", l.br("ERROR"), err.Error())
	}
}

// Logs an error and gracefully exit
func (l MaestroLog) Fatal(err error) {
	if err != nil {
		l.output.Printf("%s: %s", l.br("FATAL"), err.Error())
		HandleExit(1)
	}
}
