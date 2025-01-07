package utils

import "sync"

var instanceUIDCache struct {
	uid   string
	mutex sync.RWMutex
}

// SetInstanceUID sets the instance UID in the cache.
func SetInstanceUID(uid string) {
	instanceUIDCache.mutex.Lock()
	defer instanceUIDCache.mutex.Unlock()
	instanceUIDCache.uid = uid
}

// GetInstanceUID retrieves the instance UID from the cache.
func GetInstanceUID() string {
	instanceUIDCache.mutex.RLock()
	defer instanceUIDCache.mutex.RUnlock()
	return instanceUIDCache.uid
}
