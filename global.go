package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path"
	//"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	usageShowConsole = "use to enable the output in console"
)

var (
	pLogDir   = "."
	pHostName = "localhost"
	pPort     = "8088"
	//loggers
	infoLog  *log.Logger
	warnLog  *log.Logger
	errorLog *log.Logger
	//stats
	pStats *StatsHelper
	//signal flag
	pStillRunning = true

	pBuildTime = ""
	pVersion   = "0.1.0" + "-" + pBuildTime
	//console
	pShowConsole = true
	//envt
	pEnvVars = map[string]*string{
		"GMONGERS_LDIR": &pLogDir,
		"GMONGERS_HOST": &pHostName,
	}
)

type logOverride struct {
	Prefix string `json:"prefix,omitempty"`
}

func init() {
	//uniqueness
	rand.Seed(time.Now().UnixNano())
	//recovery
	initRecov()
	//evt
	initEnvParams()
	//loggers
	initLogger(os.Stdout, os.Stdout, os.Stderr)
	//stats
	pStats = StatsHelperNew()
	//signals
	sigHandle()
	log.Println("Ver:", pVersion)
	dumpI("Ver:", pVersion)
}

//initRecov is for dumpIng segv in
func initRecov() {
	//might help u
	defer func() {
		recvr := recover()
		if recvr != nil {
			fmt.Println("MAIN-RECOV-INIT: ", recvr)
		}
	}()
}

//os.Stdout, os.Stdout, os.Stderr
func initLogger(i, w, e io.Writer) {
	//just in case
	if !pShowConsole {
		infoLog = makeLogger(i, pLogDir, "gmongers", "INFO: ")
		warnLog = makeLogger(w, pLogDir, "gmongers", "WARN: ")
		errorLog = makeLogger(e, pLogDir, "gmongers", "ERROR: ")
	} else {
		infoLog = log.New(i,
			"INFO: ",
			log.Ldate|log.Ltime|log.Lmicroseconds)
		warnLog = log.New(w,
			"WARN: ",
			log.Ldate|log.Ltime|log.Lshortfile)
		errorLog = log.New(e,
			"ERROR: ",
			log.Ldate|log.Ltime|log.Lshortfile)
	}
}

//initEnvParams enable all OS envt vars to reload internally
func initEnvParams() {
	//just in-case, over-write from ENV
	for k, v := range pEnvVars {
		if os.Getenv(k) != "" {
			*v = os.Getenv(k)
		}
	}
	flag.BoolVar(&pShowConsole, "debug", pShowConsole, usageShowConsole)
	flag.BoolVar(&pShowConsole, "d", pShowConsole, usageShowConsole+" (shorthand)")
	flag.Parse()

}

//formatLogger try to init all filehandles for logs
func formatLogger(fdir, fname, pfx string) string {
	t := time.Now()
	r := regexp.MustCompile("[^a-zA-Z0-9]")
	p := t.Format("2006-01-02") + "-" + r.ReplaceAllString(strings.ToLower(pfx), "")
	s := path.Join(pLogDir, fdir)
	if _, err := os.Stat(s); os.IsNotExist(err) {
		//mkdir -p
		os.MkdirAll(s, os.ModePerm)
	}
	return path.Join(s, p+"-"+fname+".log")
}

//makeLogger initialize the logger either via file or console
func makeLogger(w io.Writer, ldir, fname, pfx string) *log.Logger {
	logFile := w
	if !pShowConsole {
		var err error
		logFile, err = os.OpenFile(formatLogger(ldir, fname, pfx), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
		if err != nil {
			log.Println(err)
		}
	}
	//give it
	return log.New(logFile,
		pfx,
		log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)

}

//dumpW log into warning
func dumpW(s ...interface{}) {
	warnLog.Println(s...)
}

//dumpWF log into warning w/ fmt
func dumpWF(format string, s ...interface{}) {
	warnLog.Println(fmt.Sprintf(format, s...))
}

//dumpE log into error
func dumpE(s ...interface{}) {
	errorLog.Println(s...)
}

//dumpE log into error w/ fmt
func dumpEF(format string, s ...interface{}) {
	errorLog.Println(fmt.Sprintf(format, s...))
}

//dumpI log into info
func dumpI(s ...interface{}) {
	infoLog.Println(s...)
}

//dumpIF log into info
func dumpIF(format string, s ...interface{}) {
	infoLog.Println(fmt.Sprintf(format, s...))
}
func (w logOverride) Write(bytes []byte) (int, error) {
	return fmt.Print(w.Prefix + time.Now().UTC().Format("2006-01-02 15:04:05.999") + " " + string(bytes))
}

func overrideLogger(pfx string) {
	log.SetFlags(0)
	log.SetOutput(&logOverride{Prefix: pfx})
}
