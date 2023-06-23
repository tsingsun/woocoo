package log

import (
	"log"
)

var logo = `
 ___      _______________________________ 
__ | /| / /  __ \  __ \  ___/  __ \  __ \
__ |/ |/ // /_/ / /_/ / /__ / /_/ / /_/ /
____/|__/ \____/\____/\___/ \____/\____/
`

// Println wrapper native log.Println
func Println(v ...any) {
	log.Println(v...)
}

// Printf wrapper native log.Printf
func Printf(format string, v ...any) {
	log.Printf(format, v...)
}

func PrintLogo() {
	Println(logo)
}
