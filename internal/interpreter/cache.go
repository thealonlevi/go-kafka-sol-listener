package interpreter

import (
	"fmt"
	"log"
	"sync"
)

// Reference to the SOL-to-USD cache
var solToUsdCache struct {
	rate  float64
	mutex sync.RWMutex
}

// SetSolToUsdCache allows updating the SOL-to-USD rate in the global cache.
func SetSolToUsdCache(rate float64) {
	solToUsdCache.mutex.Lock()
	defer solToUsdCache.mutex.Unlock()
	solToUsdCache.rate = rate
	log.Printf("SetSolToUsdCache: Updated SOL-to-USD rate to %.2f", rate)
}

// getCachedSolToUsdRate retrieves the current SOL-to-USD rate from the cache.
func getCachedSolToUsdRate() (float64, error) {
	solToUsdCache.mutex.RLock()
	defer solToUsdCache.mutex.RUnlock()
	if solToUsdCache.rate == 0 {
		log.Println("getCachedSolToUsdRate: SOL-to-USD rate is not available")
		return 0, fmt.Errorf("SOL-to-USD rate is not available")
	}
	log.Printf("getCachedSolToUsdRate: Retrieved SOL-to-USD rate %.2f", solToUsdCache.rate)
	return solToUsdCache.rate, nil
}
