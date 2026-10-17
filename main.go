package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"daily-word-game/db"
	"daily-word-game/game"
)

type Server struct {
	Store *db.Store
}

type GuessRequest struct {
	Player string `json:"player"`
	Guess  string `json:"guess"`
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AdminWordRequest struct {
	GameDate string `json:"game_date"`
	Word     string `json:"word"`
	Hint1    string `json:"hint_1"`
	Hint2    string `json:"hint_2"`
	Hint3    string `json:"hint_3"`
}

type FriendRequest struct {
	Player string `json:"player"`
	Friend string `json:"friend"`
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	redisURL := os.Getenv("REDIS_URL")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	store := db.NewStore(dbURL, redisURL)
	srv := &Server{Store: store}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/game", srv.handleGame)
	mux.HandleFunc("/guess", srv.handleGuess)
	mux.HandleFunc("/hints", srv.handleHints)
	mux.HandleFunc("/auth/login", srv.handleLogin)
	mux.HandleFunc("/auth/signup", srv.handleSignup)
	mux.HandleFunc("/leaderboard", srv.handleLeaderboard)
	mux.HandleFunc("/friends", srv.handleFriends)
	mux.HandleFunc("/admin/words", srv.handleAdminWords)

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fs)

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("Starting Daily Word Game server on %s ...", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pgOk := s.Store.PingPostgres()
	redisOk := s.Store.PingRedis()

	status := "ok"
	if !pgOk || !redisOk {
		status = "degraded"
	}

	resp := map[string]interface{}{
		"status":   status,
		"postgres": pgOk,
		"redis":    redisOk,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) getTodayWordAndHints() (string, string, string, string) {
	defaultWord := "CRANE"
	h1 := "First letter: C (Begins with a consonant)"
	h2 := "Vowels: Contains 2 vowels (A and E)"
	h3 := "Clue: A tall wading bird or a construction machine"

	if s.Store.DB == nil || !s.Store.PingPostgres() {
		return defaultWord, h1, h2, h3
	}

	var word, hint1, hint2, hint3 string
	query := `SELECT word, COALESCE(hint_1, ''), COALESCE(hint_2, ''), COALESCE(hint_3, '') FROM daily_words WHERE game_date = CURRENT_DATE LIMIT 1`
	err := s.Store.DB.QueryRow(query).Scan(&word, &hint1, &hint2, &hint3)
	if err != nil || strings.TrimSpace(word) == "" {
		return defaultWord, h1, h2, h3
	}

	if hint1 == "" {
		hint1 = h1
	}
	if hint2 == "" {
		hint2 = h2
	}
	if hint3 == "" {
		hint3 = h3
	}

	return strings.ToUpper(strings.TrimSpace(word)), hint1, hint2, hint3
}

func (s *Server) handleHints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, h1, h2, h3 := s.getTodayWordAndHints()
	resp := map[string]interface{}{
		"date":   time.Now().Format("2006-01-02"),
		"hint_1": h1,
		"hint_2": h2,
		"hint_3": h3,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	player := strings.TrimSpace(r.URL.Query().Get("player"))
	if player == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "player query parameter is required"})
		return
	}

	today := time.Now().Format("2006-01-02")
	targetWord, h1, h2, h3 := s.getTodayWordAndHints()

	var guessWords []string
	if s.Store.DB != nil && s.Store.PingPostgres() {
		rows, err := s.Store.DB.Query(
			`SELECT guess_word FROM guesses WHERE player_name = $1 AND game_date = CURRENT_DATE ORDER BY id ASC`,
			player,
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var gw string
				if err := rows.Scan(&gw); err == nil {
					guessWords = append(guessWords, gw)
				}
			}
		}
	}

	scoredGuesses := make([]game.ScoreResult, 0, len(guessWords))
	gameSolved := false

	for _, gw := range guessWords {
		res := game.Score(targetWord, gw)
		scoredGuesses = append(scoredGuesses, res)
		if res.Solved {
			gameSolved = true
		}
	}

	// Fetch Player Stats
	var streak, maxStreak, played, won int
	if s.Store.DB != nil && s.Store.PingPostgres() {
		_ = s.Store.DB.QueryRow(
			`SELECT current_streak, max_streak, games_played, games_won FROM players WHERE name = $1`,
			player,
		).Scan(&streak, &maxStreak, &played, &won)
	}

	resp := map[string]interface{}{
		"player":         player,
		"date":           today,
		"word_length":    len(targetWord),
		"guesses":        scoredGuesses,
		"solved":         gameSolved,
		"current_streak": streak,
		"max_streak":     maxStreak,
		"games_played":   played,
		"games_won":      won,
		"hints": map[string]string{
			"hint_1": h1,
			"hint_2": h2,
			"hint_3": h3,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleGuess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GuessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Player) == "" || strings.TrimSpace(req.Guess) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload, player and guess are required"})
		return
	}

	player := strings.TrimSpace(req.Player)
	guessWord := strings.TrimSpace(req.Guess)

	targetWord, _, _, _ := s.getTodayWordAndHints()
	scoreResult := game.Score(targetWord, guessWord)

	// Save to Postgres
	if s.Store.DB != nil && s.Store.PingPostgres() {
		// Ensure player exists
		_, _ = s.Store.DB.Exec(`INSERT INTO players (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`, player)

		_, _ = s.Store.DB.Exec(
			`INSERT INTO guesses (player_name, game_date, guess_word) VALUES ($1, CURRENT_DATE, $2)`,
			player, guessWord,
		)

		if scoreResult.Solved {
			_, _ = s.Store.DB.Exec(
				`UPDATE players SET 
					current_streak = current_streak + 1,
					max_streak = GREATEST(max_streak, current_streak + 1),
					games_played = games_played + 1,
					games_won = games_won + 1
				 WHERE name = $1`, player,
			)
		}
	}

	// Invalidate Redis cache for player board if Redis is available
	if s.Store.RDB != nil && s.Store.PingRedis() {
		cacheKey := fmt.Sprintf("board:%s:%s", player, time.Now().Format("2006-01-02"))
		s.Store.RDB.Del(context.Background(), cacheKey)
	}

	resp := map[string]interface{}{
		"player":        player,
		"guess":         scoreResult.Guess,
		"target_length": len(targetWord),
		"feedback":      scoreResult.Feedback,
		"solved":        scoreResult.Solved,
		"message":       scoreResult.Message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Username) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "username and password are required"})
		return
	}

	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)

	var role string
	var streak, maxStreak, played, won int
	if s.Store.DB != nil && s.Store.PingPostgres() {
		var dbPass string
		err := s.Store.DB.QueryRow(
			`SELECT password, role, current_streak, max_streak, games_played, games_won FROM players WHERE name = $1`,
			username,
		).Scan(&dbPass, &role, &streak, &maxStreak, &played, &won)

		if err != nil {
			// Auto create player if first login
			role = "user"
			_, _ = s.Store.DB.Exec(`INSERT INTO players (name, password) VALUES ($1, $2) ON CONFLICT DO NOTHING`, username, password)
		} else if dbPass != "" && dbPass != password {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid password"})
			return
		}
	} else {
		role = "user"
	}

	resp := map[string]interface{}{
		"status":         "success",
		"player":         username,
		"role":           role,
		"current_streak": streak,
		"max_streak":     maxStreak,
		"games_played":   played,
		"games_won":      won,
		"token":          fmt.Sprintf("token_%s_%d", username, time.Now().Unix()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	s.handleLogin(w, r)
}

func (s *Server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type LeaderItem struct {
		Rank          int    `json:"rank"`
		Name          string `json:"name"`
		GamesWon      int    `json:"games_won"`
		CurrentStreak int    `json:"current_streak"`
		GamesPlayed   int    `json:"games_played"`
	}

	var leaderboard []LeaderItem
	if s.Store.DB != nil && s.Store.PingPostgres() {
		rows, err := s.Store.DB.Query(
			`SELECT name, games_won, current_streak, games_played 
			 FROM players 
			 ORDER BY games_won DESC, current_streak DESC LIMIT 10`,
		)
		if err == nil {
			defer rows.Close()
			rank := 1
			for rows.Next() {
				var item LeaderItem
				if err := rows.Scan(&item.Name, &item.GamesWon, &item.CurrentStreak, &item.GamesPlayed); err == nil {
					item.Rank = rank
					leaderboard = append(leaderboard, item)
					rank++
				}
			}
		}
	}

	// Fallback mock if empty
	if len(leaderboard) == 0 {
		leaderboard = []LeaderItem{
			{Rank: 1, Name: "admin", GamesWon: 14, CurrentStreak: 5, GamesPlayed: 15},
			{Rank: 2, Name: "asha", GamesWon: 9, CurrentStreak: 4, GamesPlayed: 10},
			{Rank: 3, Name: "alex", GamesWon: 7, CurrentStreak: 3, GamesPlayed: 8},
			{Rank: 4, Name: "dev", GamesWon: 5, CurrentStreak: 2, GamesPlayed: 6},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"leaderboard": leaderboard})
}

func (s *Server) handleFriends(w http.ResponseWriter, r *http.Request) {
	player := strings.TrimSpace(r.URL.Query().Get("player"))
	if player == "" {
		player = "asha"
	}

	if r.Method == http.MethodPost {
		var req FriendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.Friend != "" {
			if s.Store.DB != nil && s.Store.PingPostgres() {
				_, _ = s.Store.DB.Exec(
					`INSERT INTO friends (player_name, friend_name) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
					req.Player, req.Friend,
				)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "friend added"})
		return
	}

	type FriendItem struct {
		Name   string `json:"name"`
		Streak int    `json:"streak"`
		Status string `json:"status"`
	}

	var friends []FriendItem
	if s.Store.DB != nil && s.Store.PingPostgres() {
		rows, err := s.Store.DB.Query(
			`SELECT f.friend_name, COALESCE(p.current_streak, 0)
			 FROM friends f
			 LEFT JOIN players p ON p.name = f.friend_name
			 WHERE f.player_name = $1`, player,
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var fi FriendItem
				if err := rows.Scan(&fi.Name, &fi.Streak); err == nil {
					fi.Status = "Active Today"
					friends = append(friends, fi)
				}
			}
		}
	}

	if len(friends) == 0 {
		friends = []FriendItem{
			{Name: "alex", Streak: 3, Status: "Solved Today 🎯"},
			{Name: "dev", Streak: 2, Status: "Attempting... ⏳"},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"friends": friends})
}

func (s *Server) handleAdminWords(w http.ResponseWriter, r *http.Request) {
	// Simple Admin Authentication Check
	key := r.URL.Query().Get("admin_key")
	if key != "admin123" && r.Header.Get("X-Admin-Key") != "admin123" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized: admin_key required"})
		return
	}

	if r.Method == http.MethodPost {
		var req AdminWordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Word == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload: game_date and word required"})
			return
		}

		gameDate := req.GameDate
		if gameDate == "" {
			gameDate = time.Now().Format("2006-01-02")
		}

		if s.Store.DB != nil && s.Store.PingPostgres() {
			_, err := s.Store.DB.Exec(
				`INSERT INTO daily_words (game_date, word, hint_1, hint_2, hint_3) 
				 VALUES ($1, $2, $3, $4, $5)
				 ON CONFLICT (game_date) DO UPDATE SET 
				   word = EXCLUDED.word,
				   hint_1 = EXCLUDED.hint_1,
				   hint_2 = EXCLUDED.hint_2,
				   hint_3 = EXCLUDED.hint_3`,
				gameDate, strings.ToUpper(req.Word), req.Hint1, req.Hint2, req.Hint3,
			)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "Word and 3 hints configured successfully!"})
		return
	}

	// GET: List all configured daily words and hints
	type WordItem struct {
		ID       int    `json:"id"`
		GameDate string `json:"game_date"`
		Word     string `json:"word"`
		Hint1    string `json:"hint_1"`
		Hint2    string `json:"hint_2"`
		Hint3    string `json:"hint_3"`
	}

	var words []WordItem
	if s.Store.DB != nil && s.Store.PingPostgres() {
		rows, err := s.Store.DB.Query(
			`SELECT id, to_char(game_date, 'YYYY-MM-DD'), word, COALESCE(hint_1, ''), COALESCE(hint_2, ''), COALESCE(hint_3, '') 
			 FROM daily_words ORDER BY game_date DESC LIMIT 20`,
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var item WordItem
				if err := rows.Scan(&item.ID, &item.GameDate, &item.Word, &item.Hint1, &item.Hint2, &item.Hint3); err == nil {
					words = append(words, item)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"words": words})
}
