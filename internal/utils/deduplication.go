package utils

import (
	"log"
	"sync"
)

type DeduplicationCache struct {
	cache map[string]bool
	mutex sync.RWMutex
}

// Global deduplication cache instance
var deduplicationCache = DeduplicationCache{
	cache: make(map[string]bool),
}

// AddSignature adds a signature to the cache with an initial value of `false`.
func AddSignature(signature string) {
	deduplicationCache.mutex.Lock()
	defer deduplicationCache.mutex.Unlock()
	if _, exists := deduplicationCache.cache[signature]; !exists {
		deduplicationCache.cache[signature] = false
		log.Printf("Added signature: %s to cache as unprocessed.", signature)
	} else {
		log.Printf("Signature already exists in cache: %s", signature)
	}
}

// MarkAsProcessed sets the value of a signature to `true`, indicating it has been processed.
func MarkAsProcessed(signature string) {
	deduplicationCache.mutex.Lock()
	defer deduplicationCache.mutex.Unlock()
	if _, exists := deduplicationCache.cache[signature]; exists {
		deduplicationCache.cache[signature] = true
		log.Printf("Marked signature: %s as processed.", signature)
	} else {
		log.Printf("Attempted to mark nonexistent signature as processed: %s", signature)
	}
}

// IsUnprocessed checks if a signature is in the cache and is unprocessed (`false`).
// Returns `true` if it is unprocessed, otherwise `false`.
func IsUnprocessed(signature string) bool {
	deduplicationCache.mutex.RLock()
	defer deduplicationCache.mutex.RUnlock()
	processed, exists := deduplicationCache.cache[signature]
	if !exists {
		log.Printf("Signature not found in cache: %s", signature)
		return false
	}
	log.Printf("Signature %s found in cache. Processed: %v", signature, processed)
	return !processed
}
