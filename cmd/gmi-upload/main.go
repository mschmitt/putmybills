package main

import "fmt"
import "os"
import "os/exec"
import "errors"
import "strconv"
import "path/filepath"
import "io/ioutil"
import "encoding/base64"
import "encoding/json"
import "github.com/tidwall/gjson"
import "github.com/akamensky/argparse"
import "github.com/go-resty/resty/v2"
import "verboseOutput"
import "sidecarFile"

const documentAPI string = "https://api.getmyinvoices.com/accounts/v3/documents"
var gitCommit string

func main() {
	var err error
	var envPresent bool
	var envValue string

	// Establish arguments
	parser   := argparse.NewParser("gmi-upload", "Upload document to the GetMyInvoices API")
	apikey   := parser.String("a", "apikey", &argparse.Options{Required: false, Help: "API key"})
	file     := parser.String("f", "file", &argparse.Options{Required: false, Help: "File to upload"})
	doctype  := parser.String("d", "doctype", &argparse.Options{Required: false, Default: "MISC", Help: "GMI document type"})
	docnote  := parser.String("n", "docnote", &argparse.Options{Required: false, Default: "Uploaded using https://github.com/mschmitt/putmybills", Help: "Document note"})
	resume   := parser.Flag("r", "resume", &argparse.Options{Required: false, Help: "Re-attempt dangling incomplete upload"})
	reupload := parser.Flag("R", "reupload", &argparse.Options{Required: false, Help: "Force upload of already-uploaded document"})
	verbose  := parser.Flag("v", "verbose", &argparse.Options{Required: false, Help: "Show verbose progress"})
	quiet    := parser.Flag("q", "quiet", &argparse.Options{Required: false, Help: "Don't say 'already uploaded' on previously uploaded docs"})
	version  := parser.Flag("V", "version", &argparse.Options{Required: false, Help: "Show version string (git commit)"})
	err = parser.Parse(os.Args)
	if nil != err {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}
	if true == *verbose {
		verboseOutput.Activate()
	}

	// Version requested
	if true == *version {
		fmt.Printf("Version: %s\n", gitCommit)
		os.Exit(0)
	}

	// Some arguments may be passed by environment variable:
	// GMI_APIKEY instead of -a/--apikey
	envValue, envPresent = os.LookupEnv("GMI_APIKEY")
	if envPresent == true {
		*apikey = envValue
	} else if 0 == len(*apikey) {
		fmt.Printf("ERROR: Missing option -a/--apikey or Environment GMI_APIKEY\n")
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}
	// GMI_DOCTYPE instead of -d/--doctype
	envValue, envPresent = os.LookupEnv("GMI_DOCTYPE")
	if envPresent == true {
		*doctype = envValue
	}
	// GMI_DOCNOTE instead of -n/--docnote
	envValue, envPresent = os.LookupEnv("GMI_DOCNOTE")
	if envPresent == true {
		*docnote = envValue
	}

	// Dump parameters at start of verbose operation
	verboseOutput.Out(fmt.Sprintf("%-13s: %s\n", "Commit",        gitCommit))
	verboseOutput.Out(fmt.Sprintf("%-13s: %s\n", "API token",     *apikey))
	verboseOutput.Out(fmt.Sprintf("%-13s: %s\n", "File",          *file))
	verboseOutput.Out(fmt.Sprintf("%-13s: %s\n", "Document type", *doctype))
	verboseOutput.Out(fmt.Sprintf("%-13s: %s\n", "Document note", *docnote))
	verboseOutput.Out(fmt.Sprintf("%-13s: %s\n", "Verbose",       strconv.FormatBool(*verbose)))

	// -> File not found - error message and exit != 0
	fh, err := os.OpenFile(*file, os.O_RDONLY, 0)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("ERROR: File not found: %s\n", *file)
		os.Exit(1)
	}
	fh.Close()

	// -> sidecarFile "uploading" exists - Previous upload failed without cleanup: error message and exit != 0
	_, err = sidecarFile.Read(*file, "uploading")
	if nil == err {
		if true == *resume {
			fmt.Printf("Will resume aborted upload for: %s\n", *file)
		} else {
			fmt.Printf("ERROR: Aborted upload detected for: %s (maybe retry using --resume)\n", *file)
			os.Exit(1)
		}
	}
	// -> sidecarFile "done" exists - Previous upload succeeded
	_, err = sidecarFile.Read(*file, "done")
	if nil == err {
		if true == *reupload {
			fmt.Printf("Will re-upload previously uploaded document: %s\n", *file)
		} else {
			if false == *quiet {
				fmt.Printf("File already marked as uploaded: %s\n", *file)
			}
			os.Exit(0)
		}
	}

	// Test if file is not open
	lsof := exec.Command("lsof", *file)
	err = lsof.Run()
	if nil == err {
		fmt.Printf("ERROR: File is probably open by another process: %s\n", *file)
		os.Exit(1)
	}

	// Clear existing status and set "uploading" status
	verboseOutput.Out(fmt.Sprintf("Setting sidecarFile status to \"%s\" on: %s.\n", "uploading", *file))
	sidecarFile.DeleteAny(*file)
	_, err = sidecarFile.Create(*file, "uploading", "")
	if nil != err {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(1)
	}

	// Read back to confirm that sidecar file was set
	verboseOutput.Out(fmt.Sprintf("Reading back \"%s\" upload status from: %s\n", "uploading", *file))
	_, err = sidecarFile.Read(*file, "uploading")
	if nil != err {
		fmt.Printf("ERROR: Can't read back status \"%s\" for: %s\n", "uploading", *file)
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
	sidecarFile.Delete(*file, "uploading")
	if true == uploadFailed {
		fmt.Printf("ERROR: Upload failed.\n")
		fmt.Printf("Response from API was: %s\n", response.String())
		fmt.Printf("Cleaning up.\n")
		verboseOutput.Out(fmt.Sprintf("Setting \"%s\" status on %s.\n", "failed", *file))
		sidecarFile.Create(*file, "failed", response.String())
		os.Exit(1)
	} else {
		fmt.Printf("Upload succeeded for: %s\n", *file)
		verboseOutput.Out(fmt.Sprintf("Setting \"%s\" status on: %s.\n", "done", *file))
		sidecarFile.Create(*file, "done", response.String())
		os.Exit(0)
	}
}

