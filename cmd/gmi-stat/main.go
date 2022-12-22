package main

import "fmt"
import "os"
import "github.com/pkg/xattr"

const statusAttribute string = "user.de.scsy.putmybills.upload-status"
const docUidAttribute string = "user.de.scsy.putmybills.document-uid"
const printfFormat string = "%-10s %-10s %s\n"

func main () {
	files := os.Args[1:]
	if 0 == len(files) {
		fmt.Fprintf(os.Stderr, "Usage: gmi-stat file [file]...\n")
		os.Exit(1)
	}
	fmt.Printf(printfFormat, "Status", "DocID", "Filename")
	for _, file := range files {
		var err error
		var uploadStatusBytes []byte
		var uploadStatus string
		var documentUidBytes []byte
		var documentUid string
		uploadStatusBytes, err = xattr.Get(file, statusAttribute);
		if nil != err {
			uploadStatus = "-"
		} else {
			uploadStatus = string(uploadStatusBytes)
		}
		documentUidBytes, err = xattr.Get(file, docUidAttribute);
		if nil != err {
			documentUid = "-"
		} else {
			documentUid = string(documentUidBytes)
		}
		fmt.Printf(printfFormat, uploadStatus, documentUid, file)
	}
}
