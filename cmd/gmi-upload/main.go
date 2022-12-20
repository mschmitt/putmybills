package main
import "fmt"
import "os"
import "errors"
import "strconv"
import "github.com/pkg/xattr"
import "github.com/akamensky/argparse"

// https://api.getmyinvoices.com/accounts/v3/doc/#tag/Document/operation/Upload%20new%20document
// xattrs used by this program:
// - user.putmybills.upload-status (uploading or done)
// - user.putmybills.documentUid

func main() {
	var err error
	var UploadStatusBytes []byte
	var UploadStatus string
	// Establish arguments
	parser := argparse.NewParser("gmi-upload", "Upload document to the GetMyInvoices API")
	apitoken := parser.String("a", "apitoken", &argparse.Options{Required: true, Help: "API token"})
	file     := parser.String("f", "file", &argparse.Options{Required: true, Help: "File to upload"})
	doctype  := parser.String("d", "doctype", &argparse.Options{Required: false, Default: "MISC", Help: "GMI document type"})
	resume   := parser.Flag("r", "resume", &argparse.Options{Required: false, Help: "Re-attempt dangling incomplete upload"})
	verbose  := parser.Flag("v", "verbose", &argparse.Options{Required: false, Help: "Show verbose progress"})
	err = parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}
	if true == *verbose {
		fmt.Printf("%-13s: %s\n",  "API token",    *apitoken)
		fmt.Printf("%-13s: %s\n", "File",          *file)
		fmt.Printf("%-13s: %s\n", "Document type", *doctype)
		fmt.Printf("%-13s: %s\n", "Verbose",       strconv.FormatBool(*verbose))
	}

	// -> File not found - error message and exit != 0
	fh, err := os.OpenFile(*file, os.O_RDONLY, 0644)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("ERROR: File not found: %s\n", *file)
		os.Exit(1)
	}

	// Test file for xattr capability
	XattrList, err := xattr.FList(fh);
	if err != nil {
		fmt.Printf("ERROR: Can't retrieve extended attributes for: %s\n", *file)
		fh.Close()
		os.Exit(1)
	}
	if true == *verbose {
		fmt.Printf("extended attributes for file: %s\n", *file)
		fmt.Println(XattrList)
	}

	// Get upload status xattr from file - user.putmybills.upload-status
	// error is expected at this point (empty attribute). No error handling.
	UploadStatusBytes, err = xattr.FGet(fh, "user.putmybills.upload-status");
	UploadStatus = string(UploadStatusBytes)

	// -> uploading - Previous upload failed without cleanup: error message and exit != 0
	if "uploading" == UploadStatus {
		if true == *resume {
			fmt.Printf("Will resume aborted upload for: %s\n", *file)
		} else {
			fmt.Printf("ERROR: Aborted upload detected for: %s (maybe retry using --resume)\n", *file)
			fh.Close()
			os.Exit(1)
		}
	// -> done - Previous upload succeeded, info message and abort
	} else if "done" == UploadStatus {
		fmt.Printf("File already marked as uploaded: %s\n", *file)
		fh.Close()
		os.Exit(0)
	// -> nothing - Proceed
	} else {
		if true == *verbose {
			fmt.Printf("No xattrs set. Will proceed with upload.\n")
		}
	}

	// Set upload status xattr: uploading
	err = xattr.FSet(fh, "user.putmybills.upload-status", []byte("uploading"))
	if err != nil {
		fmt.Printf("ERROR: Can't set extended attributes for: %s\n", *file)
		fh.Close()
		os.Exit(1)
	}

	// Ready to upload
	fh.Close()

	// Upload to API

	// Test for success: 1) HTTP 200, success = true, documentUid defined

	// Set upload documentUid xattr (tbd: name of xattr)
	// Set upload status xattr: done
	// Message to user: success, blah
	// exit 0

	// If upload failed: 1) HTTP != 200, success != true, documentUid undefined
	// Unset upload status xattr
	// Dump JSON if received from server
	// Error message and exit != 0
}

