package verboseOutput
import "fmt"

var wantOutput = false;

func Activate() {
	wantOutput = true;
}
func Dectivate() {
	wantOutput = false;
}

func Out(message string) {
	if true == wantOutput {
		fmt.Printf("VERBOSE: %s", message)
	}
}
