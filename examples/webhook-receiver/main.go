package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// DecodedLog represents the structure sent by the scanner
type DecodedLog struct {
	EventName   string                 `json:"event_name"`
	Address     string                 `json:"address"`
	BlockNumber uint64                 `json:"block_number"`
	TxHash      string                 `json:"tx_hash"`
	Data        map[string]interface{} `json:"data"`
}

func main() {
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading body", http.StatusInternalServerError)
			return
		}

		var logs []DecodedLog
		if err := json.Unmarshal(body, &logs); err != nil {
			fmt.Printf("Received raw body: %s\n", string(body))
		} else {
			fmt.Printf("Received %d events via webhook:\n", len(logs))
			for _, l := range logs {
				fmt.Printf(" - [%s] Tx: %s | Event: %s\n", l.Address, l.TxHash, l.EventName)
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fmt.Println("Webhook receiver listening on :8080/webhook...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
