# Daily Word Game 🎯

A high-performance Wordle-style daily word game microservice built in **Go 1.22**, containerized with **Docker & Docker Compose**, backed by **PostgreSQL** and **Redis**, and integrated with a full **CircleCI CI/CD pipeline**.

---

## 1. Duplicate Letters Scoring Algorithm (Answer to "The Hard Part")

### The Problem
In Wordle-style games, words frequently contain duplicate characters. When scoring a guess against a target word, simple per-character lookups produce incorrect feedback (e.g. marking extra guess letters as yellow when all instances in the target word have already been claimed).

### Contract Example
* **Target Word**: `CRANE` (Length 5: `C:1`, `R:1`, `A:1`, `N:1`, `E:1`)
* **Guess Word**: `EERIE` (Length 5: `E` at pos 0, 1, 4; `R` at pos 2; `I` at pos 3)

Notice that target `CRANE` contains **exactly ONE `E`**, while guess `EERIE` contains **three `E`s**.

### Scoring Rules
1. **Greens Claim First**: Exact index matches (same letter at same position) are evaluated **first** and marked `GREEN`. These matching target characters are claimed immediately and removed from the pool of available target letters.
2. **Yellow Matching**: Unmatched guess positions are evaluated left-to-right. If the character exists in the remaining pool of unclaimed target letters, it is marked `YELLOW`, and the available count for that character is decremented by 1.
3. **Gray (Excess / Absent)**: Any remaining guess characters that do not match or exceed the remaining available count in the target pool are marked `GRAY`.

### Walkthrough: `CRANE` (Target) vs `EERIE` (Guess)

```
Target:  C  R  A  N  E
Guess:   E  E  R  I  E
Index:   0  1  2  3  4
```

#### Pass 1 — Green Evaluation:
* Index 0: `E` vs `C` -> Mismatch. Target `C` added to available pool.
* Index 1: `E` vs `R` -> Mismatch. Target `R` added to available pool.
* Index 2: `R` vs `A` -> Mismatch. Target `A` added to available pool.
* Index 3: `I` vs `N` -> Mismatch. Target `N` added to available pool.
* Index 4: `E` vs `E` -> **MATCH!** Index 4 is **`GREEN`**. Target `E` is claimed.

**Available Target Pool after Pass 1**: `{ C: 1, R: 1, A: 1, N: 1 }` *(Notice `E` count is **0**)*.

#### Pass 2 — Yellow & Gray Evaluation:
* Index 0 (`E`): `E` count in pool is 0 -> **`GRAY`** (excess `E`).
* Index 1 (`E`): `E` count in pool is 0 -> **`GRAY`** (excess `E`).
* Index 2 (`R`): `R` count in pool is 1 -> **`YELLOW`** (`R` pool count decremented to 0).
* Index 3 (`I`): `I` not in pool -> **`GRAY`**.
* Index 4 (`E`): Already **`GREEN`**.

#### Final Output for `EERIE`:
* Pos 0 `E`: `GRAY`
* Pos 1 `E`: `GRAY`
* Pos 2 `R`: `YELLOW`
* Pos 3 `I`: `GRAY`
* Pos 4 `E`: `GREEN`

**Result**: Exactly **ONE `E`** is colored (the `GREEN` at index 4). The other two `E`s are `GRAY` because `GREEN` claimed the only `E` first.

---

## 2. Quick Start with Docker Compose

Start the application, PostgreSQL, Redis, and run automated database migrations with a single command:

```bash
docker compose up --build
```

### Health Check Verification
Once started, run:

```bash
curl localhost:8080/health
```

Expected Output:
```json
{"postgres":true,"redis":true,"status":"ok"}
```

---

## 3. Application Endpoints

### 1. Health Check
* **`GET /health`**
* Reports database & cache connectivity status independently.
* Example Response:
  ```json
  {
    "status": "ok",
    "postgres": true,
    "redis": true
  }
  ```

### 2. Get Game Board
* **`GET /game?player=asha`**
* Returns today's active word game board and historical guesses for the specified player.
* Example Response:
  ```json
  {
    "date": "2026-10-09",
    "guesses": [
      {
        "guess": "EERIE",
        "feedback": [
          {"letter": "E", "status": "GRAY"},
          {"letter": "E", "status": "GRAY"},
          {"letter": "R", "status": "YELLOW"},
          {"letter": "I", "status": "GRAY"},
          {"letter": "E", "status": "GREEN"}
        ],
        "solved": false
      }
    ],
    "player": "asha",
    "solved": false,
    "word_length": 5
  }
  ```

### 3. Submit Guess
* **`POST /guess`**
* Payload: `{"player": "asha", "guess": "CRANE"}`
* Example Response:
  ```json
  {
    "feedback": [
      {"letter": "C", "status": "GREEN"},
      {"letter": "R", "status": "GREEN"},
      {"letter": "A", "status": "GREEN"},
      {"letter": "N", "status": "GREEN"},
      {"letter": "E", "status": "GREEN"}
    ],
    "guess": "CRANE",
    "player": "asha",
    "solved": true,
    "target_length": 5
  }
  ```

---

## 4. Running Unit & Integration Tests

### Unit Tests
Run unit tests for game scoring logic (100% coverage):

```bash
docker run --rm -v "${PWD}:/app" -w /app golang:1.22-alpine go test ./game/... -cover
```

### Integration Tests
Integration tests execute against real PostgreSQL and Redis containers. Environment variables:
* `DATABASE_URL`: `postgres://postgres:postgres@localhost:5432/wordgame?sslmode=disable`
* `REDIS_URL`: `redis://localhost:6379`

```bash
go test -v ./...
```

---

## 5. CI/CD Pipeline (.circleci/config.yml)

The CircleCI configuration enforces a strict quality gate:

1. **`lint`**: Code formatting (`gofmt`) and static analysis (`go vet`).
2. **`unit-tests`**: Runs fast logic tests for `game/score.go`.
3. **`integration-tests`**: Spins up real PostgreSQL 16 and Redis 7 service containers, executes `migrations/*.sql` in filename order, and runs HTTP integration tests against `localhost`.
4. **`secret-scan`**: Scans codebase for hardcoded private keys or access tokens.
5. **`image-build`**: Builds the production Docker container image, tags with version `:1.0` and `:latest`, and pushes to Docker Hub.

