package gen

import (
	"fmt"
)

// Expect panics if the condition is false.
func Expect(cond bool, msg string, args ...any) {
	if !cond {
		panic(GraphError{fmt.Sprintf(msg, args...)})
	}
}

func CheckGraphError(err error, msg string, args ...any) {
	if err != nil {
		args = append(args, err)
		panic(GraphError{fmt.Sprintf(msg+": %s", args...)})
	}
}

type GraphError struct {
	msg string
}

func (p GraphError) Error() string { return fmt.Sprintf("entc/gen: %s", p.msg) }

func CatchGraphError(err *error) {
	if e := recover(); e != nil {
		gerr, ok := e.(GraphError)
		if !ok {
			panic(e)
		}
		*err = gerr
	}
}
