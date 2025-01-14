// webhook.go

package interpreter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// sendToWebhook sends the JSON data **double-encoded** to the webhook URL.
func sendToWebhook(details map[string]interface{}, webhookURL string) (*http.Response, error) {
	// 1) Marshal your enriched details into a JSON string.
	innerJSON, err := json.Marshal(details)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal enriched details: %w", err)
	}

	// 2) Build a top-level object with a "body" field containing that JSON string.
	topLevel := map[string]string{
		"body": string(innerJSON),
	}

	// 3) Marshal the top-level object.
	finalPayload, err := json.Marshal(topLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to double-encode payload: %w", err)
	}

	// 4) POST the double-encoded JSON to the webhook.
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(finalPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to send enriched details to webhook: %w", err)
	}

	return resp, nil
}
