package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"daily-word-game/db"
)

func TestIntegration_Endpoints(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	redisURL := os.Getenv("REDIS_URL")

	if dbURL == "" || redisURL == "" {
		t.Log("DATABASE_URL or REDIS_URL not set; running endpoint unit/integration tests with store fallback")
	}

	store := db.NewStore(dbURL, redisURL)
	srv := &Server{Store: store}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/game", srv.handleGame)
	mux.HandleFunc("/guess", srv.handleGuess)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Run("GET /health Endpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/health")
		if err != nil {
			t.Fatalf("failed to GET /health: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		var healthRes map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&healthRes); err != nil {
			t.Fatalf("failed to decode health response: %v", err)
		}

		if dbURL != "" && redisURL != "" {
			if healthRes["postgres"] != true || healthRes["redis"] != true {
				t.Errorf("expected true for postgres and redis when DBs are configured, got %v", healthRes)
			}
			if healthRes["status"] != "ok" {
				t.Errorf("expected status 'ok', got %v", healthRes["status"])
			}
		}
	})

	t.Run("GET /game Validation", func(t *testing.T) {
		// Missing player query parameter
		resp, err := http.Get(ts.URL + "/game")
		if err != nil {
			t.Fatalf("failed to GET /game: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for missing player, got %d", resp.StatusCode)
		}

		// Valid player query
		respPlayer, err := http.Get(ts.URL + "/game?player=asha")
		if err != nil {
			t.Fatalf("failed to GET /game?player=asha: %v", err)
		}
		defer respPlayer.Body.Close()

		if respPlayer.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK for valid player, got %d", respPlayer.StatusCode)
		}
	})

	t.Run("POST /guess and game progression", func(t *testing.T) {
		playerName := "test_player_" + time.Now().Format("150405")

		// 1. Invalid payload
		badResp, err := http.Post(ts.URL+"/guess", "application/json", bytes.NewBufferString("{}"))
		if err != nil {
			t.Fatalf("failed to POST /guess: %v", err)
		}
		badResp.Body.Close()
		if badResp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for empty payload, got %d", badResp.StatusCode)
		}

		// 2. Guess duplicate letters (EERIE against CRANE)
		guessBody, _ := json.Marshal(map[string]string{
			"player": playerName,
			"guess":  "EERIE",
		})
		resp, err := http.Post(ts.URL+"/guess", "application/json", bytes.NewBuffer(guessBody))
		if err != nil {
			t.Fatalf("failed to POST /guess: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}

		var guessRes map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&guessRes); err != nil {
			t.Fatalf("failed to decode JSON: %v", err)
		}

		if guessRes["player"] != playerName {
			t.Errorf("expected player %s, got %v", playerName, guessRes["player"])
		}

		feedback, ok := guessRes["feedback"].([]interface{})
		if !ok || len(feedback) != 5 {
			t.Fatalf("expected 5 feedback items, got %v", guessRes["feedback"])
		}

		// 3. Winning guess (CRANE)
		winBody, _ := json.Marshal(map[string]string{
			"player": playerName,
			"guess":  "CRANE",
		})
		winResp, err := http.Post(ts.URL+"/guess", "application/json", bytes.NewBuffer(winBody))
		if err != nil {
			t.Fatalf("failed to POST winning guess: %v", err)
		}
		defer winResp.Body.Close()

		var winRes map[string]interface{}
		json.NewDecoder(winResp.Body).Decode(&winRes)
		if winRes["solved"] != true {
			t.Errorf("expected solved=true for winning guess, got %v", winRes["solved"])
		}
	})
}
