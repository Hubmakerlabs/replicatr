// Package log is a logging subsystem that provides code optional location tracing and semi-automated subsystem registration and output control.
package log

import (
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
)

const (
	Off Level = iota
	Fatal
	Error
	Check
	Warn
	Info
	Debug
	Trace
)

// gLS is a helper to make more compact declarations of LevelSpec names by using
// the Level LvlStr map.
func gLS(lvl Level) LevelSpec {
	return LevelSpec{
		Name: LvlStr[lvl],
	}
}

var (
	// LevelSpecs specifies the id, string name and color-printing function
	LevelSpecs = map[Level]LevelSpec{
		Off:   gLS(Off),
		Fatal: gLS(Fatal),
		Error: gLS(Error),
		Check: gLS(Check),
		Warn:  gLS(Warn),
		Info:  gLS(Info),
		Debug: gLS(Debug),
		Trace: gLS(Trace),
	}

	// LvlStr is a map that provides the uniform width strings that are printed
	// to identify the Level of a log entry.
	LvlStr = LevelMap{
		Off:   "Off",
		Fatal: "Fatal",
		Error: "Error",
		Warn:  "Warn",
		Info:  "Info",
		Check: "Check",
		Debug: "Debug",
		Trace: "Trace",
	}

	// LvlStrShort is a map for compact versions for use in the printer.
	LvlStrShort = LevelMap{
		Off:   "",
		Fatal: "FTL",
		Error: "ERR",
		Warn:  "WRN",
		Info:  "INF",
		Check: "CHK",
		Debug: "DBG",
		Trace: "TRC",
	}

	// log is your generic Log creation invocation that uses the version data
	// in version.go that provides the current compilation path prefix for making
	// relative paths for log printing code locations.
	lvlStrs = map[string]Level{
		"off": Off,
		"ftl": Fatal,
		"err": Error,
		"chk": Check,
		"wrn": Warn,
		"inf": Info,
		"dbg": Debug,
		"trc": Trace,
	}
	levelMx  sync.Mutex
	logLevel = Info
)

type (
	LevelMap map[Level]string
	// Level is a code representing a scale of importance and context for log
	// entries.
	Level int32
	// Println prints lists of interfaces with spaces in between
	Println func(a ...interface{})
	// Printf prints like fmt.Println surrounded by log details
	Printf func(format string, a ...interface{})
	// Prints  prints a spew.Sdump for an interface slice
	Prints func(a ...interface{})
	// Printc accepts a function so that the extra computation can be avoided if
	// it is not being viewed
	Printc func(closure func() string)
	// Chk is a shortcut for printing if there is an error, or returning true
	Chk func(e error) bool
	// LevelPrinter defines a set of terminal printing primitives that output
	// with extra data, time, level, and code location
	LevelPrinter struct {
		Ln Println
		// F prints like fmt.Println surrounded by log details
		F Printf
		// S uses spew.dump to show the content of a variable
		S Prints
		// C accepts a function so that the extra computation can be avoided if
		// it is not being viewed
		C Printc
		// Chk is a shortcut for printing if there is an error, or returning
		// true
		Chk Chk
	}
	// LevelSpec is a key pair of log level and the text colorizer used
	// for it.
	LevelSpec struct {
		Name string
	}
	// Log is a set of log printers for the various Level items.
	Log struct {
		F, E, W, I, D, T LevelPrinter
	}
)

// GetLoc calls runtime.Caller to get the path of the calling source code file.
func GetLoc(skip int) (output string) {
	_, file, line, _ := runtime.Caller(skip)
	output = fmt.Sprint(file, ":", line)
	return
}

// New returns a set of LevelPrinter with their subsystem preloaded
//
// this copies the interface of stdlib log but we don't respect the settings
// because a logger without timestamps is retarded
func New(writer io.Writer, appID string, _ int) (l *Log) {
	return &Log{
		getOnePrinter(writer, appID, Fatal),
		getOnePrinter(writer, appID, Error),
		getOnePrinter(writer, appID, Warn),
		getOnePrinter(writer, appID, Info),
		getOnePrinter(writer, appID, Debug),
		getOnePrinter(writer, appID, Trace),
	}
}

func SetLogLevel(l Level) {
	levelMx.Lock()
	defer levelMx.Unlock()
	logLevel = l
}

func GetLogLevel() (l Level) {
	levelMx.Lock()
	defer levelMx.Unlock()
	l = logLevel
	return
}

func (l LevelMap) String() (s string) {
	ss := make([]string, len(l))
	for i := range l {
		ss[i] = strings.TrimSpace(l[i])
	}
	return strings.Join(ss, " ")
}

func _c(writer io.Writer, appID string, level Level) Printc {
	return func(closure func() string) {
		logPrint(writer, appID, level, closure)()
	}
}
func _chk(writer io.Writer, appID string, level Level) Chk {
	return func(e error) (is bool) {
		if e != nil {
			logPrint(writer, appID, level,
				joinStrings(
					" ",
					"CHECK:",
					e,
				))()
			is = true
		}
		return
	}
}
func _f(writer io.Writer, appID string, level Level) Printf {
	return func(format string, a ...interface{}) {
		logPrint(writer, appID,
			level, func() string {
				return fmt.Sprintf(format, a...)
			},
		)()
	}
}
func _ln(writer io.Writer, appID string, l Level) Println {
	return func(a ...interface{}) {
		logPrint(writer, appID, l,
			backticksToSingleQuote(joinStrings(" ", a...)()))()
	}
}
func _s(writer io.Writer, appID string, level Level) Prints {
	return func(a ...interface{}) {
		text := "spew:\n"
		if s, ok := a[0].(string); ok {
			text = strings.TrimSpace(s) + "\n"
			a = a[1:]
		}
		logPrint(writer, appID,
			level, func() string {
				return backticksToSingleQuote(text + spew.Sdump(a...))()
			},
		)()
	}
}

func backticksToSingleQuote(in string) (out func() string) {
	return func() string {
		return strings.ReplaceAll(in, "`", "'")
	}
}

func getOnePrinter(writer io.Writer, appID string, level Level) LevelPrinter {
	return LevelPrinter{
		Ln:  _ln(writer, appID, level),
		F:   _f(writer, appID, level),
		S:   _s(writer, appID, level),
		C:   _c(writer, appID, level),
		Chk: _chk(writer, appID, level),
	}
}

// getTimeText is a helper that returns the current time with the
// timeStampFormat that is configured.
func getTimeText(tsf string) string { return time.Now().Format(tsf) }

// joinStrings constructs a string from a slice of interface same as Println but
// without the terminal newline
func joinStrings(sep string, a ...interface{}) func() (o string) {
	return func() (o string) {
		for i := range a {
			o += fmt.Sprint(a[i])
			if i < len(a)-1 {
				o += sep
			}
		}
		return
	}
}

// UnixNanoAsFloat e
func UnixNanoAsFloat() (s string) {
	timeText := fmt.Sprint(time.Now().UnixNano())
	lt := len(timeText)
	lb := lt + 1
	var timeBytes = make([]byte, lb)
	copy(timeBytes[lb-9:lb], timeText[lt-9:lt])
	timeBytes[lb-10] = '.'
	lb -= 10
	lt -= 9
	copy(timeBytes[:lb], timeText[:lt])
	return string(timeBytes)
}

var formatString = "%s" +
	" " +
	"%s" +
	" " +
	"%s" +
	" " +
	"`%s`" +
	" " +
	"%s" +
	"\n"

// logPrint is the generic log printing function that provides the base
// format for log entries.
func logPrint(writer io.Writer, appID string, level Level,
	printFunc func() string) func() {

	return func() {
		levelMx.Lock()
		defer levelMx.Unlock()
		if level > logLevel {
			return
		}
		s := fmt.Sprintf(
			formatString,
			UnixNanoAsFloat(),
			"["+appID+"]",
			LvlStrShort[level],
			printFunc(),
			GetLoc(3),
		)
		_, _ = fmt.Fprint(writer, s)
	}
}
