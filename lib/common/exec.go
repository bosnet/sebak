package common

import (
	"bytes"
	"os/exec"
)

// wrapExternalCommand is for the other platform like windows. SEBAK does not
// support officially, and if you wish to support windows platform, create
// custom `wrapExternalCommand` for it.
func wrapExternalCommand(cmd string) (string, []string) {
	return "sh", []string{"-c", cmd}
}

func ExecExternalCommand(cmd string) ([]byte, error) {
	name, extras := wrapExternalCommand(cmd)

	c := exec.Command(name, extras...)

	var b bytes.Buffer
	c.Stderr = &b
	c.Stdout = &b

	err := c.Run()
	return b.Bytes(), err
}
