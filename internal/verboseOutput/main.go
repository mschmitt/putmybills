package verboseOutput
import "fmt"

type verboseOut struct {
	enabled bool
}

func (v *verboseOut) Enable() {
	v.enabled = true;
}

func (v *verboseOut) Disable() {
	v.enabled = false;
}

func New(enabled bool) verboseOut {
	return verboseOut{enabled: enabled}
}

func (v *verboseOut) Out(message string) {
	if true ==  v.enabled {
		fmt.Printf("VERBOSE: %s", message)
	}
}
