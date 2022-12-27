package sidecarFile

import "fmt"
import "os"
import "errors"
import "path/filepath"

func deriveSidecar(baseFile string, sidecarExt string) (sidecarFile string, errormsg error) {
	// https://stackoverflow.com/a/12518877
	if _, err := os.Stat(baseFile); errors.Is(err, os.ErrNotExist) {
		sidecarFile = "/dev/null"
		errormsg = errors.New(fmt.Sprintf("baseFile %s not found.", baseFile))
		return
	}
	sidecarFile = fmt.Sprintf("%s.%s", baseFile, sidecarExt)
	return
}

func Create(baseFile string, sidecarExt string, content string) (sidecarFile string, errormsg error){
	sidecarFile, err := deriveSidecar(baseFile, sidecarExt)
	if nil != err {
		errormsg = err
		return
	}
	fh, err := os.OpenFile(sidecarFile, os.O_RDWR|os.O_CREATE, 0644)
	if nil != err {
		errormsg = errors.New(fmt.Sprintf("Can't open sidecarFile for writing: %s\n", sidecarFile))
		return
	}
	_, err = fh.WriteString(content)
	if nil != err {
		fh.Close()
		errormsg = errors.New(fmt.Sprintf("Can't write to file: %s\n", sidecarFile))
		return
	}
	fh.Close()
	return
}

func Read(baseFile string, sidecarExt string) (content string, errormsg error){
	sidecarFile, err := deriveSidecar(baseFile, sidecarExt)
	if nil != err {
		errormsg = err
		return
	}
	bytecontent, err := os.ReadFile(sidecarFile)
	if nil != err {
		errormsg = errors.New(fmt.Sprintf("Can't open sidecarFile for reading: %s\n", sidecarFile))
		return
	}
	content = string(bytecontent)
	return
}

func Delete(baseFile string, sidecarExt string) (errormsg error){
	sidecarFile, err := deriveSidecar(baseFile, sidecarExt)
	if nil != err {
		errormsg = err
		return
	}
	err = os.Remove(sidecarFile)
	if nil != err {
		errormsg = err
		return
	}
	return
}

func DeleteAny(baseFile string) (errormsg error){
	// https://stackoverflow.com/a/48073701/263310
	sidecarFiles, err := filepath.Glob(fmt.Sprintf("%s.*", baseFile))
	if nil != err {
		errormsg = err
		return
	}
	for _, sidecarFile := range sidecarFiles {
		err = os.Remove(sidecarFile)
		if nil != err {
			errormsg = err
			return
		}
	}
	return
}
