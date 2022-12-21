package main

import "fmt"
import "os"
import "errors"
import "strconv"
import "github.com/tidwall/gjson"
import "github.com/pkg/xattr"
import "github.com/akamensky/argparse"
import "github.com/go-resty/resty/v2"

// https://api.getmyinvoices.com/accounts/v3/doc/#tag/Document/operation/Upload%20new%20document

const statusAttribute string = "user.de.scsy.putmybills.upload-status"
const docUidAttribute string = "user.de.scsy.putmybills.document-uid"
const documentAPI string = "https://api.getmyinvoices.com/accounts/v3/documents"

func main() {
	var err error
	var uploadStatusBytes []byte
	var uploadStatus string

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
	fh.Close()

	// Test file for xattr capability
	XattrList, err := xattr.List(*file);
	if err != nil {
		fmt.Printf("ERROR: Can't retrieve extended attributes for: %s\n", *file)
		os.Exit(1)
	}
	if true == *verbose {
		fmt.Printf("extended attributes for file: %s\n", *file)
		fmt.Println(XattrList)
	}

	// Get upload status xattr from file - user.putmybills.upload-status
	// - Error is expected at this point (empty attribute). 
	// - No error handling.
	uploadStatusBytes, err = xattr.Get(*file, statusAttribute);
	uploadStatus = string(uploadStatusBytes)

	// -> uploading - Previous upload failed without cleanup: error message and exit != 0
	if "uploading" == uploadStatus {
		if true == *resume {
			fmt.Printf("Will resume aborted upload for: %s\n", *file)
		} else {
			fmt.Printf("ERROR: Aborted upload detected for: %s (maybe retry using --resume)\n", *file)
			os.Exit(1)
		}
	// -> done - Previous upload succeeded, info message and abort
	} else if "done" == uploadStatus {
		fmt.Printf("File already marked as uploaded: %s\n", *file)
		os.Exit(0)
	// -> nothing - Proceed
	} else {
		if true == *verbose {
			fmt.Printf("No xattrs set. Will proceed with upload.\n")
		}
	}

	// Set upload status xattr: uploading
	if true == *verbose {
		fmt.Printf("Setting %s xattr to \"%s\" on: %s.\n", statusAttribute, "uploading", *file)
	}
	err = xattr.Set(*file, statusAttribute, []byte("uploading"))
	if err != nil {
		fmt.Printf("ERROR: Can't set extended attributes for: %s\n", *file)
		os.Exit(1)
	}

	// Read back to confirm that attribute was set
	if true == *verbose {
		fmt.Printf("Reading back %s xattr (expecting: \"%s\") from: %s\n", statusAttribute, "uploading", *file)
	}
	uploadStatusBytes, err = xattr.Get(*file, statusAttribute);
	uploadStatus = string(uploadStatusBytes)
	if err != nil {
		fmt.Printf("ERROR: Can't read back attribute %s for: %s\n", statusAttribute, *file)
		os.Exit(1)
	}

	// Ready to upload

	// Upload to API
	client := resty.New()
	// Dummy API interaction (Get list of documents)
	response, err := client.R().
		EnableTrace().
		SetHeader("Content-Type", "application/json").
		SetHeader("X-API-KEY", *apitoken).
		Get(documentAPI)
	if err != nil {
		fmt.Printf("ERROR: HTTP request to %s failed: %s", documentAPI, err)
		os.Exit(1)
	}

	// Early abort if HTTP status is not 200
	if true == *verbose {
		fmt.Printf("Checking HTTP status.\n")
	}
	if response.StatusCode() != 200 {
		fmt.Printf("ERROR: HTTP request to %s failed: %s\n", documentAPI, response.Status())
		os.Exit(1)
	}

	// Analyze response
	var uploadFailed bool = false;
	if true == *verbose {
		fmt.Printf("Looking for \"success\" in response.\n")
	}
	success := gjson.Get(response.String(), "success")
	if success.Type.String() == "Null" {
		fmt.Printf("ERROR: No success reported.\n")
		uploadFailed = true;
	}

	if true == *verbose {
		fmt.Printf("Looking for \"documentUid\" in response.\n")
	}
	documentUid := gjson.Get(response.String(), "documentUid")
	if documentUid.Type.String() == "Null" {
		fmt.Printf("ERROR: No documentUid reported.\n")
		uploadFailed = true;
	}

	if uploadFailed == true {
		fmt.Printf("Upload failed.\n")
		fmt.Printf("Cleaning up.\n")
		if true == *verbose {
			fmt.Printf("Deleting %s xattr on: %s.\n", statusAttribute, *file)
		}
		xattr.Remove(*file, statusAttribute)
		os.Exit(1)
	} else {
		fmt.Printf("Upload succeeded.\n")
		if true == *verbose {
			fmt.Printf("Setting %s xattr to \"%s\" on: %s.\n", statusAttribute, "done", *file)
		}
		err = xattr.Set(*file, statusAttribute, []byte("done"))
		if err != nil {
			fmt.Printf("ERROR: Can't set extended attributes for: %s\n", *file)
			os.Exit(1)
		}
		if true == *verbose {
			fmt.Printf("Setting %s xattr to \"%s\" on: %s.\n", docUidAttribute, documentUid.String(), *file)
		}
		err = xattr.Set(*file, docUidAttribute, []byte(documentUid.String()))
		if err != nil {
			fmt.Printf("ERROR: Can't set extended attributes for: %s\n", *file)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

