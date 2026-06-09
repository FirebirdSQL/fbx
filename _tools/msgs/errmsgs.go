package main

import (
	"fmt"
	"os"
)

// To rebuild messages: go run ./_tools/msgs fbproto/errmsgs.go

//#include "msgs.h"
import "C"
import (
	"strings"
)

var outFile *os.File
var sbMessages strings.Builder
var sbCodes strings.Builder

func writeOut(val string) {
	_, err := outFile.WriteString(val)
	if err != nil {
		fmt.Printf("Unable to write file: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	var err error

	if len(os.Args) < 2 {
		fmt.Println("Output file required")
		os.Exit(1)
	}
	if outFile, err = os.OpenFile(os.Args[1], os.O_CREATE|os.O_RDWR, 0644); err != nil {
		fmt.Printf("Unable to open file: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()
	C.process_messages()

	writeOut(`// Code generated from _tools/msgs DO NOT EDIT.
package fbproto

type fberror struct {
	message  string
	sqlstate string
}

`)

	writeOut("const (\n")
	writeOut(sbCodes.String())
	writeOut(")\n\n")

	writeOut("var errmsgs = map[int]fberror{\n")
	writeOut(sbMessages.String())
	writeOut("}\n")
}

//export addMessage
func addMessage(code C.int, message *C.char, class *C.char, subClass *C.char, symbol *C.char) {
	var msg string = C.GoString(message)
	var cl string = C.GoString(class)
	var sub string = C.GoString(subClass)
	var sym string = C.GoString(symbol)
	name := fmt.Sprintf("Isc_%s", sym)

	if !strings.HasSuffix(msg, `\n"`) {
		msg = msg[:len(msg)-1]
		msg += `\n"`
	}
	sbMessages.WriteString(fmt.Sprintf("\t%s: {%s, \"%s%s\"},\n", name, msg, cl, sub))
	sbCodes.WriteString(fmt.Sprintf("\t%s = %d\n", name, int(code)))
}
