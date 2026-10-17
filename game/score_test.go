package game_test

import (
	"reflect"
	"testing"

	"daily-word-game/game"
)

func TestScore_DuplicateLetters(t *testing.T) {
	// Key test requirement from prompt:
	// Answer: CRANE, Guess: EERIE
	// There is exactly ONE E in CRANE.
	// Guess has three E's (indices 0, 1, 4).
	// Green claims index 4 first. Indices 0 and 1 receive GRAY because green claimed the only E.
	target := "CRANE"
	guess := "EERIE"

	result := game.Score(target, guess)

	expectedStatuses := []game.Status{
		game.StatusGray,   // E (pos 0) - excess occurrence
		game.StatusGray,   // E (pos 1) - excess occurrence
		game.StatusYellow, // R (pos 2) - present in CRANE at index 1
		game.StatusGray,   // I (pos 3) - absent
		game.StatusGreen,  // E (pos 4) - exact match with CRANE at index 4
	}

	if len(result.Feedback) != len(expectedStatuses) {
		t.Fatalf("expected feedback length %d, got %d", len(expectedStatuses), len(result.Feedback))
	}

	for i, expected := range expectedStatuses {
		if result.Feedback[i].Status != expected {
			t.Errorf("at index %d for letter %s: expected status %s, got %s",
				i, result.Feedback[i].Letter, expected, result.Feedback[i].Status)
		}
	}

	// Count how many E's are colored (GREEN or YELLOW)
	coloredECount := 0
	for _, fb := range result.Feedback {
		if fb.Letter == "E" && (fb.Status == game.StatusGreen || fb.Status == game.StatusYellow) {
			coloredECount++
		}
	}

	if coloredECount != 1 {
		t.Errorf("expected exactly 1 colored 'E', got %d", coloredECount)
	}

	if result.Solved {
		t.Errorf("expected Solved to be false, got true")
	}
}

func TestScore_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		target         string
		guess          string
		expectedSolved bool
		expectedStatus []game.Status
	}{
		{
			name:           "Exact Match",
			target:         "CRANE",
			guess:          "CRANE",
			expectedSolved: true,
			expectedStatus: []game.Status{
				game.StatusGreen, game.StatusGreen, game.StatusGreen, game.StatusGreen, game.StatusGreen,
			},
		},
		{
			name:           "All Incorrect",
			target:         "CRANE",
			guess:          "BOOST",
			expectedSolved: false,
			expectedStatus: []game.Status{
				game.StatusGray, game.StatusGray, game.StatusGray, game.StatusGray, game.StatusGray,
			},
		},
		{
			name:           "Mixed Yellow and Green",
			target:         "CRANE",
			guess:          "CANES",
			expectedSolved: false,
			expectedStatus: []game.Status{
				game.StatusGreen,  // C
				game.StatusYellow, // A
				game.StatusYellow, // N
				game.StatusYellow, // E
				game.StatusGray,   // S
			},
		},
		{
			name:           "Multiple Target Duplicates",
			target:         "SPEED",
			guess:          "ERASE",
			expectedSolved: false,
			expectedStatus: []game.Status{
				game.StatusYellow, // E
				game.StatusGray,   // R
				game.StatusGray,   // A
				game.StatusYellow, // S
				game.StatusYellow, // E
			},
		},
		{
			name:           "Lowercase and spaces normalization",
			target:         " crane ",
			guess:          " eerie ",
			expectedSolved: false,
			expectedStatus: []game.Status{
				game.StatusGray, game.StatusGray, game.StatusYellow, game.StatusGray, game.StatusGreen,
			},
		},
		{
			name:           "Length Mismatch",
			target:         "CRANE",
			guess:          "WORD",
			expectedSolved: false,
			expectedStatus: []game.Status{
				game.StatusGray, game.StatusGray, game.StatusGray, game.StatusGray,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := game.Score(tt.target, tt.guess)
			if res.Solved != tt.expectedSolved {
				t.Errorf("expected solved %v, got %v", tt.expectedSolved, res.Solved)
			}

			statuses := make([]game.Status, len(res.Feedback))
			for i, fb := range res.Feedback {
				statuses[i] = fb.Status
			}

			if !reflect.DeepEqual(statuses, tt.expectedStatus) {
				t.Errorf("expected statuses %v, got %v", tt.expectedStatus, statuses)
			}
		})
	}
}
