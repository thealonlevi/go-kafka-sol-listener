// python_executor.go

package interpreter

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

// invokePythonScript executes the Python script and returns the result of the swap detection.
func invokePythonScript(jsonData []byte) (string, error) {
	log.Println("Hello")
	cmd := exec.Command(pythonInterpreter, swapDetectorScript)
	cmd.Stdin = bytes.NewBuffer(jsonData)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute Python script: %v\nOutput: %s", err, output.String())
	}
	return output.String(), nil
}
