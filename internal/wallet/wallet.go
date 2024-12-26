package wallet

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type WalletManager struct {
	walletMap map[string]bool
	mutex     sync.RWMutex
	url       string
}

func NewWalletManager(url string) *WalletManager {
	return &WalletManager{
		walletMap: make(map[string]bool),
		url:       url,
	}
}

// UpdateWallets fetches the wallet list and updates the map
func (wm *WalletManager) UpdateWallets() {
	for {
		resp, err := http.Get(wm.url)
		if err != nil {
			log.Printf("Failed to fetch wallet list: %v\n", err)
			time.Sleep(1 * time.Minute)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Non-OK HTTP status: %s\n", resp.Status)
			time.Sleep(1 * time.Minute)
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read wallet list response: %v\n", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		var walletList []string
		if err := json.Unmarshal(body, &walletList); err != nil {
			log.Printf("Failed to parse wallet list: %v\n", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		// Update the wallet map
		newWalletMap := make(map[string]bool)
		for _, wallet := range walletList {
			newWalletMap[wallet] = true
		}

		wm.mutex.Lock()
		wm.walletMap = newWalletMap
		wm.mutex.Unlock()

		log.Println("Wallet list updated successfully.")

		// Print all wallets in the updated wallet list
		log.Println("Current wallet list:")
		for wallet := range newWalletMap {
			log.Println(wallet)
		}

		time.Sleep(1 * time.Minute)
	}
}

// WalletExists checks if a wallet exists in the map
func (wm *WalletManager) WalletExists(wallet string) bool {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()
	return wm.walletMap[wallet]
}

// GetWalletList retrieves the current wallet list as a slice of strings
func (wm *WalletManager) GetWalletList() []string {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()

	wallets := make([]string, 0, len(wm.walletMap))
	for wallet := range wm.walletMap {
		wallets = append(wallets, wallet)
	}
	return wallets
}
