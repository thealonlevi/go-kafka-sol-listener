// signature.go

package interpreter

// extractSignature extracts the transaction signature from the message.
func extractSignature(message map[string]interface{}) (string, bool) {
	transaction, ok := message["Transaction"].(map[string]interface{})
	if !ok {
		return "", false
	}

	signature, ok := transaction["Signature"].(string)
	return signature, ok
}
