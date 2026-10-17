-- Migration 003: Advanced Features Schema (Auth, Stats, Hints, Friends, Admin)

-- 1. Extend Players table with Auth and Stats
ALTER TABLE players 
ADD COLUMN IF NOT EXISTS password VARCHAR(255) DEFAULT 'password123',
ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT 'user',
ADD COLUMN IF NOT EXISTS current_streak INT DEFAULT 0,
ADD COLUMN IF NOT EXISTS max_streak INT DEFAULT 0,
ADD COLUMN IF NOT EXISTS games_played INT DEFAULT 0,
ADD COLUMN IF NOT EXISTS games_won INT DEFAULT 0;

-- 2. Extend Daily Words table with 3 Hints
ALTER TABLE daily_words 
ADD COLUMN IF NOT EXISTS hint_1 VARCHAR(255) DEFAULT 'First letter: C',
ADD COLUMN IF NOT EXISTS hint_2 VARCHAR(255) DEFAULT 'Vowels: Contains A and E',
ADD COLUMN IF NOT EXISTS hint_3 VARCHAR(255) DEFAULT 'Meaning: A tall wading bird or lifting machine';

-- Update today's word CRANE with explicit hints
UPDATE daily_words 
SET hint_1 = 'First letter: C (Begins with a consonant)',
    hint_2 = 'Vowels: Contains 2 vowels (A and E)',
    hint_3 = 'Clue: A tall wading bird or a construction machine'
WHERE game_date = CURRENT_DATE;

-- Insert future words for next few days
INSERT INTO daily_words (game_date, word, hint_1, hint_2, hint_3)
VALUES 
  (CURRENT_DATE + INTERVAL '1 day', 'SMART', 'First letter: S', 'Vowels: Contains 1 vowel (A)', 'Clue: Clever or intelligent'),
  (CURRENT_DATE + INTERVAL '2 day', 'PLANT', 'First letter: P', 'Vowels: Contains 1 vowel (A)', 'Clue: Flora or to place seeds in soil'),
  (CURRENT_DATE + INTERVAL '3 day', 'FLAME', 'First letter: F', 'Vowels: Contains 2 vowels (A and E)', 'Clue: Hot glowing body of ignited gas'),
  (CURRENT_DATE + INTERVAL '4 day', 'SHINE', 'First letter: S', 'Vowels: Contains 2 vowels (I and E)', 'Clue: To give out or reflect light')
ON CONFLICT (game_date) DO NOTHING;

-- 3. Friends Table
CREATE TABLE IF NOT EXISTS friends (
    id SERIAL PRIMARY KEY,
    player_name VARCHAR(100) NOT NULL,
    friend_name VARCHAR(100) NOT NULL,
    status VARCHAR(20) DEFAULT 'accepted',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_friendship UNIQUE (player_name, friend_name)
);

-- Seed initial admin user and sample friends
INSERT INTO players (name, password, role, current_streak, max_streak, games_played, games_won)
VALUES 
  ('admin', 'admin123', 'admin', 5, 12, 15, 14),
  ('alex', 'password123', 'user', 3, 5, 8, 7),
  ('dev', 'password123', 'user', 2, 4, 6, 5)
ON CONFLICT (name) DO NOTHING;

INSERT INTO friends (player_name, friend_name)
VALUES 
  ('asha', 'alex'),
  ('asha', 'dev'),
  ('alex', 'asha')
ON CONFLICT DO NOTHING;
