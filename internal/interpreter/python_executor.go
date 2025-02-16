// Canvas: python_executor.go

package interpreter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
)

// invokePythonScript executes the Python script and returns the result of the swap detection.
// The "jsonData" parameter is your base transaction/swap details in JSON format.
func invokePythonScript(jsonData []byte) (string, error) {
	log.Println("Hello")

	// 1. Fetch SOL-to-USD rate
	rate, err := getCachedSolToUsdRate()
	if err != nil {
		log.Printf("Failed to fetch SOL-to-USD rate: %v", err)
		// You might handle the error differently, but let's allow zero or a fallback if the rate isn't found
		rate = 0.0
	}

	log.Println("Sending rate: ", rate)

	// 2. Unmarshal your existing JSON data into a map
	var baseData map[string]interface{}
	if err := json.Unmarshal(jsonData, &baseData); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	// 3. Add the SOL-USD rate into that map
	baseData["solUsdRate"] = rate
	log.Println(baseData["Transaction"])

	// 4. Marshal the updated data
	updatedJSON, err := json.Marshal(baseData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal combined JSON data: %v", err)
	}

	// 5. Invoke the python script with the updated JSON

	cmd := exec.Command(pythonInterpreter, swapDetectorScript)
	cmd.Stdin = bytes.NewBuffer(updatedJSON)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute Python script: %v\nOutput: %s", err, output.String())
	}

	return output.String(), nil
}
