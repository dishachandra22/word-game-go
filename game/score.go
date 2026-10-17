package game

import "strings"

type Status string

const (
	StatusGreen  Status = "GREEN"  // Correct letter in correct position
	StatusYellow Status = "YELLOW" // Correct letter in wrong position
	StatusGray   Status = "GRAY"   // Letter not present or excess occurrence
)

type LetterFeedback struct {
	Letter string `json:"letter"`
	Status Status `json:"status"`
}

type ScoreResult struct {
	Guess    string           `json:"guess"`
	Feedback []LetterFeedback `json:"feedback"`
	Solved   bool             `json:"solved"`
	Message  string           `json:"message"`
}

// Score evaluates a guess against a target word using Wordle duplicate letter rules.
// Rule: Greens claim letter matches first. Remaining occurrences in target are matched left-to-right for Yellows.
// Any excess occurrences in guess beyond target letter counts are marked Gray.
func Score(target, guess string) ScoreResult {
	targetUpper := strings.ToUpper(strings.TrimSpace(target))
	guessUpper := strings.ToUpper(strings.TrimSpace(guess))

	targetRunes := []rune(targetUpper)
	guessRunes := []rune(guessUpper)

	n := len(guessRunes)
	feedback := make([]LetterFeedback, n)

	if len(targetRunes) != n {
		for i, r := range guessRunes {
			feedback[i] = LetterFeedback{
				Letter: string(r),
				Status: StatusGray,
			}
		}
		return ScoreResult{
			Guess:    guessUpper,
			Feedback: feedback,
			Solved:   false,
			Message:  "Word length mismatch",
		}
	}

	targetCounts := make(map[rune]int)
	greenMatches := make([]bool, n)

	// Pass 1: Mark Green matches and collect count of non-green target letters
	for i := 0; i < n; i++ {
		if guessRunes[i] == targetRunes[i] {
			greenMatches[i] = true
			feedback[i] = LetterFeedback{
				Letter: string(guessRunes[i]),
				Status: StatusGreen,
			}
		} else {
			targetCounts[targetRunes[i]]++
		}
	}

	// Pass 2: Mark Yellow for available target letters, or Gray for remaining
	solved := true
	for i := 0; i < n; i++ {
		if greenMatches[i] {
			continue
		}
		solved = false
		r := guessRunes[i]
		if count, exists := targetCounts[r]; exists && count > 0 {
			feedback[i] = LetterFeedback{
				Letter: string(r),
				Status: StatusYellow,
			}
			targetCounts[r]--
		} else {
			feedback[i] = LetterFeedback{
				Letter: string(r),
				Status: StatusGray,
			}
		}
	}

	message := "Keep trying! You have attempts remaining."
	if solved {
		message = "Splendid! You guessed the daily word!"
	}

	return ScoreResult{
		Guess:    guessUpper,
		Feedback: feedback,
		Solved:   solved,
		Message:  message,
	}
}

//demo 3
