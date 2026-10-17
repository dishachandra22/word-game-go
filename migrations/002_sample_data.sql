INSERT INTO daily_words (game_date, word)
VALUES (CURRENT_DATE, 'CRANE')
ON CONFLICT (game_date) DO UPDATE SET word = 'CRANE';

INSERT INTO players (name)
VALUES ('asha'), ('bob')
ON CONFLICT (name) DO NOTHING;
