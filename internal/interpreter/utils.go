// utils.go
package interpreter

import (
	"go-kafka-sol-listener/internal/utils"
	"log"
)

// getInstanceUID fetches the current instance UID from the utils package.
func getInstanceUID() string {
	uid := utils.GetInstanceUID()
	log.Println("utils.go: Instance UID: ", uid)
	return uid
}
