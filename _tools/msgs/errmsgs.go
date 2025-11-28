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

	writeOut(`package fbproto

type fberror struct {
	message  string
	sqlstate string
}

var errmsgs = map[int]fberror{
`)

	C.process_messages()
	writeOut("}\n")
}

//export addMessage
func addMessage(code C.int, message *C.char, class *C.char, subClass *C.char) {
	var msg string = C.GoString(message)
	var cl string = C.GoString(class)
	var sub string = C.GoString(subClass)
	if !strings.HasSuffix(msg, `\n"`) {
		msg = msg[:len(msg)-1]
		msg += `\n"`
	}
	writeOut(fmt.Sprintf("\t%d: {%s, \"%s%s\"},\n", int(code), msg, cl, sub))
}
