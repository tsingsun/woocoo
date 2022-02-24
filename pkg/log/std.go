package log

import (
	"flag"
	"log"
)

var logo = `
 ___      _______________________________ 
__ | /| / /  __ \  __ \  ___/  __ \  __ \
__ |/ |/ // /_/ / /_/ / /__ / /_/ / /_/ /
____/|__/ \____/\____/\___/ \____/\____/
`

//StdPrintln use native log.Println
func StdPrintln(v ...interface{}) {
	log.Println(v...)
}

func StdPrintf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func PrintLogo() {
	if !isTestRun() {
		StdPrintln(logo)
	}
}

func isTestRun() bool {
	flag.Parse()
	t := flag.Lookup("test.v")
	if t == nil {
		return false
	}
	return t.Value.(flag.Getter).Get().(bool)
}
