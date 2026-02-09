package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/otiai10/namazu/internal/delivery/webhook"
)

const testSecret = "test-secret-key"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	http.HandleFunc("/webhook", handleWebhook)

	log.Printf("Test webhook server starting on http://localhost:%s/webhook", port)
	log.Println("Secret:", testSecret)
	log.Println("Waiting for earthquake notifications...")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify signature
	signature := r.Header.Get("X-Signature-256")
	if !webhook.Verify(testSecret, body, signature) {
		log.Println("⚠️  Invalid signature!")
	} else {
		log.Println("✓ Signature verified")
	}

	// Handle URL verification challenge
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err == nil {
		if payload["type"] == "url_verification" {
			challenge, _ := payload["challenge"].(string)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"challenge": challenge})
			log.Printf("URL verification challenge responded: %s", challenge)
			return
		}

		// Pretty print JSON for normal events
		formatted, _ := json.MarshalIndent(payload, "", "  ")
		log.Printf("\n=== Earthquake Received at %s ===\n%s\n",
			time.Now().Format("15:04:05"), string(formatted))
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
