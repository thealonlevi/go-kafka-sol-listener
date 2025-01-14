// logging.go

package interpreter

import (
	"encoding/json"
	"log"
)

// logEnrichedDetails neatly prints the enriched swap details in JSON format.
func logEnrichedDetails(details map[string]interface{}) {
	enrichedJSON, err := json.MarshalIndent(details, "", "  ")
	if err != nil {
		log.Printf("Failed to format enriched details: %v\n", err)
		return
	}
	log.Printf("Enriched Swap Details:\n%s\n", string(enrichedJSON))
}