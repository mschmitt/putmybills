package main

import "fmt"
import "os"
import "errors"
import "strconv"
import "path/filepath"
import "io/ioutil"
import "encoding/base64"
import "encoding/json"
import "github.com/tidwall/gjson"
import "github.com/pkg/xattr"
import "github.com/akamensky/argparse"
import "github.com/go-resty/resty/v2"
import "verboseOutput"

const statusAttribute string = "user.de.scsy.putmybills.upload-status"
const docUidAttribute string = "user.de.scsy.putmybills.document-uid"
const documentAPI string = "https://api.getmyinvoices.com/accounts/v3/documents"

func main() {
	var err error
	var uploadStatusBytes []byte
	var uploadStatus string

	// Establish arguments
	parser   := argparse.NewParser("gmi-upload", "Upload document to the GetMyInvoices API")
	apikey   := parser.String("a", "apikey", &argparse.Options{Required: true, Help: "API key"})
	file     := parser.String("f", "file", &argparse.Options{Required: true, Help: "File to upload"})
	doctype  := parser.String("d", "doctype", &argparse.Options{Required: false, Default: "MISC", Help: "GMI document type"})
	docnote  := parser.String("n", "docnote", &argparse.Options{Required: false, Default: "Uploaded using https://github.com/mschmitt/putmybills", Help: "Document note"})
	resume   := parser.Flag("r", "resume", &argparse.Options{Required: false, Help: "Re-attempt dangling incomplete upload"})
	reupload := parser.Flag("R", "reupload", &argparse.Options{Required: false, Help: "Force upload of already-uploaded document"})
	verbose  := parser.Flag("v", "verbose", &argparse.Options{Required: false, Help: "Show verbose progress"})
	err = parser.Parse(os.Args)
	if nil != err {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}
	if true == *verbose {
		verboseOutput.Activate()
	}
	verboseOutput.Out(fmt.Sprintf("%-13s: %s\n",  "API token",    *apikey))
	verboseOutput.Out(fmt.Sprintf("%-13s: %s\n", "File",          *file))
	verboseOutput.Out(fmt.Sprintf("%-13s: %s\n", "Document type", *doctype))
	verboseOutput.Out(fmt.Sprintf("%-13s: %s\n", "Verbose",       strconv.FormatBool(*verbose)))

	// -> File not found - error message and exit != 0
	fh, err := os.OpenFile(*file, os.O_RDONLY, 0644)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("ERROR: File not found: %s\n", *file)
		os.Exit(1)
	}
	fh.Close()

	// Test file for xattr capability
	XattrList, err := xattr.List(*file);
	if nil != err {
		fmt.Printf("ERROR: Can't retrieve extended attributes for: %s\n", *file)
		os.Exit(1)
	}
	verboseOutput.Out(fmt.Sprintf("extended attributes for file: %s\n", *file))
	verboseOutput.Out(fmt.Sprintln(XattrList))

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
	// -> done - Previous upload succeeded
	} else if "done" == uploadStatus {
		if true == *reupload {
			fmt.Printf("Will re-upload previously uploaded document: %s\n", *file)
		} else {
			fmt.Printf("File already marked as uploaded: %s\n", *file)
			os.Exit(0)
		}
	// -> nothing - Proceed
	} else {
		verboseOutput.Out(fmt.Sprintf("No xattrs set. Will proceed with upload.\n"))
	}

	// Set upload status xattr: uploading
	verboseOutput.Out(fmt.Sprintf("Setting %s xattr to \"%s\" on: %s.\n", statusAttribute, "uploading", *file))
	err = xattr.Set(*file, statusAttribute, []byte("uploading"))
	if nil != err {
		fmt.Printf("ERROR: Can't set extended attributes for: %s\n", *file)
		os.Exit(1)
	}

	// Read back to confirm that attribute was set
	verboseOutput.Out(fmt.Sprintf("Reading back %s xattr (expecting: \"%s\") from: %s\n", statusAttribute, "uploading", *file))
	uploadStatusBytes, err = xattr.Get(*file, statusAttribute);
	uploadStatus = string(uploadStatusBytes)
	if nil != err {
		fmt.Printf("ERROR: Can't read back attribute %s for: %s\n", statusAttribute, *file)
		os.Exit(1)
	}

	// Ready to upload
	var uploadFailed bool = false;

	// Read file into memory
	fileBytes, err := ioutil.ReadFile(*file)
	if nil != err {
		fmt.Printf("ERROR: Failed to read file: %s\n", err)
		os.Exit(1)
	}
	fileBase64 := base64.StdEncoding.EncodeToString(fileBytes)

	// Build JSON object for upload
	gmiPayloadData := map[string]interface{}{
		"fileName": filepath.Base(*file),
		"documentType": *doctype,
		"fileContent": fileBase64,
		"note": *docnote,
	}
	gmiPayload, err := json.Marshal(gmiPayloadData)
	if nil != err {
		fmt.Printf("ERROR: json encoding failed: %s\n", err)
		os.Exit(1)
	}
	verboseOutput.Out(fmt.Sprintf("%+v\n", string(gmiPayload)))

	// Upload to API
	client := resty.New()
	response, err := client.R().
		EnableTrace().
		SetHeader("Content-Type", "application/json").
		SetHeader("X-API-KEY", *apikey).
		SetBody(gmiPayload).
		Post(documentAPI)
	if nil != err {
		fmt.Printf("ERROR: HTTP request to %s failed: %s", documentAPI, err)
		uploadFailed = true;
	}

	// Analyze response

	// HTTP status is 200?
	verboseOutput.Out(fmt.Sprintf("Checking HTTP status.\n"))
	if response.StatusCode() != 200 {
		fmt.Printf("ERROR: HTTP request to %s failed (HTTP status %d != 200).\n", documentAPI, response.StatusCode())
		uploadFailed = true;
	}

	// Success?
	verboseOutput.Out(fmt.Sprintf("Looking for \"success\" in response.\n"))
	success := gjson.Get(response.String(), "success")
	if success.Type.String() == "Null" {
		fmt.Printf("ERROR: No success reported (not even false).\n")
		uploadFailed = true;
	} else if success.Bool() != true {
		fmt.Printf("ERROR: success false reported.\n")
		uploadFailed = true;
	}

	// Got documentUid?
	verboseOutput.Out(fmt.Sprintf("Looking for \"documentUid\" in response.\n"))
	documentUid := gjson.Get(response.String(), "documentUid")
	if documentUid.Type.String() == "Null" {
		fmt.Printf("ERROR: No documentUid reported.\n")
		uploadFailed = true;
	}

	// Conclusion
	if true == uploadFailed {
		fmt.Printf("ERROR: Upload failed.\n")
		fmt.Printf("Response from API was: %s\n", response.String())
		fmt.Printf("Cleaning up.\n")
		verboseOutput.Out(fmt.Sprintf("Deleting %s xattr on: %s.\n", statusAttribute, *file))
		xattr.Remove(*file, statusAttribute)
		os.Exit(1)
	} else {
		fmt.Printf("Upload succeeded.\n")
		verboseOutput.Out(fmt.Sprintf("Setting %s xattr to \"%s\" on: %s.\n", statusAttribute, "done", *file))
		err = xattr.Set(*file, statusAttribute, []byte("done"))
		if nil != err {
			fmt.Printf("ERROR: Can't set extended attributes for: %s\n", *file)
			os.Exit(1)
		}
		verboseOutput.Out(fmt.Sprintf("Setting %s xattr to \"%s\" on: %s.\n", docUidAttribute, documentUid.String(), *file))
		err = xattr.Set(*file, docUidAttribute, []byte(documentUid.String()))
		if nil != err {
			fmt.Printf("ERROR: Can't set extended attributes for: %s\n", *file)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

