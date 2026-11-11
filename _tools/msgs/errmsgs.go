package main

import "C"
import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// To rebuild error messages:   go run ./_tools/msgs msg fbproto/errmsgs.go
// To rebuild builtin tz list:  go run ./_tools/msgs tz fbproto/builtin_tz.go

//#include "msgs.h"
import "C"

var (
	outFile    *os.File
	sbMessages strings.Builder
	sbCodes    strings.Builder
	paramRegex = regexp.MustCompile(`@\d+`)
)

func writeOut(val string) {
	_, err := outFile.WriteString(val)
	if err != nil {
		fmt.Printf("Unable to write file: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Printf("Usage: msgs <msg|tz> <file>\n")
}

func main() {
	var err error

	if len(os.Args) < 3 {
		usage()
		os.Exit(1)
	}

	genType := os.Args[1]
	if genType != "msg" && genType != "tz" {
		usage()
		os.Exit(1)
	}

	if outFile, err = os.OpenFile(os.Args[2], os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644); err != nil {
		fmt.Printf("Unable to open file: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	if genType == "msg" {
		genMsgs()
	} else {
		genTz()
	}
}

func genMsgs() {
	C.process_messages()

	writeOut(`// Code generated from _tools/msgs DO NOT EDIT.
package fbproto

type fberror struct {
	message  string
	params   int
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

func genTz() {
	tzFile, err := os.OpenFile("_tools/msgs/include/firebird/TimeZones.h", os.O_RDONLY, 0644)
	if err != nil {
		fmt.Printf("Unable to open file: %v\n", err)
		os.Exit(1)
	}
	defer tzFile.Close()

	writeOut(`// Code generated from _tools/msgs/include/firebird/TimeZones.h DO NOT EDIT.
package fbproto

var timezoneNameByID = map[int]string{
`)

	r := regexp.MustCompile(`#define\s+fb_tzid_.*\s+(\d+)\s+\/\*\s+([^\s]+)\s+\*\/`)

	scanner := bufio.NewScanner(tzFile)
	for scanner.Scan() {
		line := scanner.Text()
		match := r.FindStringSubmatch(line)
		if match != nil {
			writeOut(fmt.Sprintf("\t%s: \"%s\",\n", match[1], match[2]))
		} else {
			fmt.Printf("Skipping line '%s'\n", line)
		}
	}
	err = scanner.Err()
	if err != nil {
		fmt.Printf("Unable to read file file: %v\n", err)
		os.Exit(1)
	}
	writeOut("}\n")
}

//export addMessage
func addMessage(code C.int, message *C.char, class *C.char, subClass *C.char, symbol *C.char) {
	var msg string = C.GoString(message)
	var cl string = C.GoString(class)
	var sub string = C.GoString(subClass)
	var sym string = C.GoString(symbol)
	name := fmt.Sprintf("%d", int(code))
	if sym != "" {
		name = fmt.Sprintf("Isc_%s", sym)
		sbCodes.WriteString(fmt.Sprintf("\t%s = %d\n", name, int(code)))
	}
	msg = strings.TrimRight(msg, "\n")
	sbMessages.WriteString(fmt.Sprintf("\t%s: {%s, %d, \"%s%s\"},\n", name, fmt.Sprintf("%#v", msg), countParameters(msg), cl, sub))
}

func countParameters(msg string) int {
	return len(paramRegex.FindAllString(msg, -1))
}
