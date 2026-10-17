/**
 * Daily Word Game - Ultimate Full Stack Platform Client Engine
 */

document.addEventListener('DOMContentLoaded', () => {
  const WORD_LENGTH = 5;
  const MAX_ATTEMPTS = 6;

  let currentPlayer = localStorage.getItem('wordgame_player') || 'asha';
  let currentAttempt = 0;
  let currentGuess = '';
  let isGameOver = false;
  let isSubmitting = false;
  let activeGameGuesses = [];

  // Key state tracking: 'A' => 'GREEN' | 'YELLOW' | 'GRAY'
  const keyStates = {};

  // Hints storage
  let currentHints = {
    hint_1: 'First letter: C (Begins with a consonant)',
    hint_2: 'Vowels: Contains 2 vowels (A and E)',
    hint_3: 'Clue: A tall wading bird or a construction machine'
  };

  // DOM Elements
  const gridBoard = document.getElementById('grid-board');
  const playerInput = document.getElementById('player-input');
  const switchPlayerBtn = document.getElementById('switch-player-btn');
  const activeUserDisplay = document.getElementById('active-user-display');
  const statusText = document.getElementById('status-text');
  const toastContainer = document.getElementById('toast-container');
  const attemptBadge = document.getElementById('attempt-badge');
  const attemptProgressBar = document.getElementById('attempt-progress-bar');
  const countdownTimer = document.getElementById('countdown-timer');

  // Modals & Buttons
  const hintsBtn = document.getElementById('hints-btn');
  const hintsModal = document.getElementById('hints-modal');
  const closeHintsBtn = document.getElementById('close-hints-btn');

  const leaderboardBtn = document.getElementById('leaderboard-btn');
  const leaderboardModal = document.getElementById('leaderboard-modal');
  const closeLeaderboardBtn = document.getElementById('close-leaderboard-btn');
  const leaderboardBody = document.getElementById('leaderboard-body');

  const friendsBtn = document.getElementById('friends-btn');
  const friendsModal = document.getElementById('friends-modal');
  const closeFriendsBtn = document.getElementById('close-friends-btn');
  const friendsList = document.getElementById('friends-list');
  const addFriendBtn = document.getElementById('add-friend-btn');
  const friendNameInput = document.getElementById('friend-name-input');

  const authBtn = document.getElementById('auth-btn');
  const authModal = document.getElementById('auth-modal');
  const closeAuthBtn = document.getElementById('close-auth-btn');
  const authForm = document.getElementById('auth-form');

  const adminBtn = document.getElementById('admin-btn');
  const adminModal = document.getElementById('admin-modal');
  const closeAdminBtn = document.getElementById('close-admin-btn');
  const adminWordForm = document.getElementById('admin-word-form');
  const adminWordsBody = document.getElementById('admin-words-body');

  const statsBtn = document.getElementById('stats-btn');
  const statsModal = document.getElementById('stats-modal');
  const closeStatsBtn = document.getElementById('close-stats-btn');
  const shareResultsBtn = document.getElementById('share-results-btn');

  // Stats Elements
  const statPlayed = document.getElementById('stat-played');
  const statWinRate = document.getElementById('stat-win-rate');
  const statStreak = document.getElementById('stat-streak');
  const statMaxStreak = document.getElementById('stat-max-streak');
  const gameStatusSummary = document.getElementById('game-status-summary');

  // 1. Initialize Grid
  function initGrid() {
    gridBoard.innerHTML = '';
    for (let r = 0; r < MAX_ATTEMPTS; r++) {
      const row = document.createElement('div');
      row.className = 'grid-row';
      row.id = `row-${r}`;
      
      for (let c = 0; c < WORD_LENGTH; c++) {
        const tile = document.createElement('div');
        tile.className = 'tile';
        tile.id = `tile-${r}-${c}`;

        const tileInner = document.createElement('div');
        tileInner.className = 'tile-inner';

        const tileFront = document.createElement('div');
        tileFront.className = 'tile-face tile-front';

        const tileBack = document.createElement('div');
        tileBack.className = 'tile-face tile-back';

        tileInner.appendChild(tileFront);
        tileInner.appendChild(tileBack);
        tile.appendChild(tileInner);

        row.appendChild(tile);
      }
      gridBoard.appendChild(row);
    }
  }

  function parseStatus(fbItem) {
    if (!fbItem) return 'GRAY';
    if (typeof fbItem === 'string') return fbItem.toUpperCase();
    if (typeof fbItem === 'object' && fbItem.status) return String(fbItem.status).toUpperCase();
    return 'GRAY';
  }

  // 2. Fetch & Sync Game State
  async function loadGameState(player) {
    currentPlayer = player;
    localStorage.setItem('wordgame_player', player);
    playerInput.value = player;
    activeUserDisplay.innerHTML = `Logged in as: <strong>${player}</strong>`;
    
    currentAttempt = 0;
    currentGuess = '';
    isGameOver = false;
    isSubmitting = false;
    activeGameGuesses = [];
    resetKeyboard();
    initGrid();
    updateAttemptIndicator();

    try {
      const res = await fetch(`/game?player=${encodeURIComponent(player)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      
      statusText.textContent = 'Postgres & Redis Synced';
      
      if (data.hints) {
        currentHints = data.hints;
        updateHintsUI();
      }

      if (data.guesses && Array.isArray(data.guesses)) {
        activeGameGuesses = data.guesses;
        data.guesses.forEach((guessRes, index) => {
          renderScoredRow(index, guessRes, false);
          currentAttempt++;
        });
        
        updateAttemptIndicator();

        if (data.solved) {
          isGameOver = true;
          updateStatsUI(data.current_streak, data.max_streak, data.games_played, data.games_won, true);
        } else if (currentAttempt >= MAX_ATTEMPTS) {
          isGameOver = true;
          updateStatsUI(data.current_streak, data.max_streak, data.games_played, data.games_won, false);
        } else {
          updateStatsUI(data.current_streak, data.max_streak, data.games_played, data.games_won, false);
        }
      }
    } catch (err) {
      console.error('Failed to load game state:', err);
      statusText.textContent = 'Backend Offline';
    }
  }

  // 3. Render Scored Row
  function renderScoredRow(rowIndex, scoreResult, animate = true) {
    const row = document.getElementById(`row-${rowIndex}`);
    if (!row) return;

    const guess = (scoreResult.guess || '').toUpperCase();
    const feedback = scoreResult.feedback || [];

    for (let c = 0; c < WORD_LENGTH; c++) {
      const tile = document.getElementById(`tile-${rowIndex}-${c}`);
      if (!tile) continue;

      const tileFront = tile.querySelector('.tile-front');
      const tileBack = tile.querySelector('.tile-back');
      const letter = guess[c] || '';
      const status = parseStatus(feedback[c]);
      const statusClass = status.toLowerCase();

      if (animate) {
        tileFront.textContent = letter;
        tileBack.textContent = letter;
        tileBack.className = `tile-face tile-back ${statusClass}`;

        setTimeout(() => {
          tile.classList.add('flipped');
          updateKeyState(letter, status);
        }, c * 250);
      } else {
        tileFront.textContent = letter;
        tileBack.textContent = letter;
        tileBack.className = `tile-face tile-back ${statusClass}`;
        tile.classList.add('flipped');
        updateKeyState(letter, status);
      }
    }
  }

  function updateKeyState(letter, status) {
    if (!letter) return;
    const char = letter.toUpperCase();
    
    if (keyStates[char] === 'GREEN') return;
    if (keyStates[char] === 'YELLOW' && status === 'GRAY') return;

    keyStates[char] = status;
    const keyBtn = document.querySelector(`.key[data-key="${char}"]`);
    if (keyBtn) {
      keyBtn.classList.remove('green', 'yellow', 'gray');
      keyBtn.classList.add(status.toLowerCase());
    }
  }

  function resetKeyboard() {
    Object.keys(keyStates).forEach(k => delete keyStates[k]);
    document.querySelectorAll('.key').forEach(btn => {
      btn.classList.remove('green', 'yellow', 'gray');
    });
  }

  // 4. Submit Guess
  async function submitGuess() {
    if (isGameOver || isSubmitting) return;
    if (currentGuess.length < WORD_LENGTH) {
      shakeRow(currentAttempt);
      showToast('Word must be 5 letters!', 1800);
      return;
    }

    isSubmitting = true;
    const guessToSubmit = currentGuess.toUpperCase();

    try {
      const response = await fetch('/guess', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ player: currentPlayer, guess: guessToSubmit })
      });

      if (!response.ok) throw new Error('Failed to submit guess');

      const data = await response.json();
      activeGameGuesses.push(data);
      renderScoredRow(currentAttempt, data, true);

      const totalAnim = (WORD_LENGTH - 1) * 250 + 600;
      setTimeout(() => {
        if (data.solved) {
          isGameOver = true;
          bounceWinningRow(currentAttempt);
          triggerVictoryEffects();
        } else {
          currentAttempt++;
          currentGuess = '';
          updateAttemptIndicator();

          if (currentAttempt >= MAX_ATTEMPTS) {
            isGameOver = true;
            showToast(`Game Over! Maximum 6 attempts reached today.`, 4000);
          }
        }
        isSubmitting = false;
      }, totalAnim);

    } catch (err) {
      console.error(err);
      shakeRow(currentAttempt);
      showToast('Error communicating with server.', 2500);
      isSubmitting = false;
    }
  }

  function handleInputKey(key) {
    if (isGameOver || isSubmitting) return;
    const char = key.toUpperCase();

    if (char === 'ENTER') {
      submitGuess();
    } else if (char === 'BACKSPACE' || char === 'DELETE') {
      if (currentGuess.length > 0) {
        currentGuess = currentGuess.slice(0, -1);
        updateActiveRowUI();
      }
    } else if (/^[A-Z]$/.test(char)) {
      if (currentGuess.length < WORD_LENGTH) {
        currentGuess += char;
        updateActiveRowUI();
      }
    }
  }

  function updateActiveRowUI() {
    for (let c = 0; c < WORD_LENGTH; c++) {
      const tile = document.getElementById(`tile-${currentAttempt}-${c}`);
      if (!tile) continue;

      const tileFront = tile.querySelector('.tile-front');
      if (c < currentGuess.length) {
        const letter = currentGuess[c];
        if (tileFront.textContent !== letter) {
          tileFront.textContent = letter;
          tile.classList.add('pop');
          setTimeout(() => tile.classList.remove('pop'), 150);
        }
        tile.classList.add('active-filled');
      } else {
        tileFront.textContent = '';
        tile.classList.remove('active-filled', 'pop');
      }
    }
  }

  function shakeRow(rowIndex) {
    const row = document.getElementById(`row-${rowIndex}`);
    if (row) {
      row.classList.add('shake');
      setTimeout(() => row.classList.remove('shake'), 500);
    }
  }

  function bounceWinningRow(rowIndex) {
    const row = document.getElementById(`row-${rowIndex}`);
    if (row) {
      row.querySelectorAll('.tile').forEach((tile, idx) => {
        setTimeout(() => tile.classList.add('bounce'), idx * 100);
      });
    }
  }

  function updateAttemptIndicator() {
    if (attemptBadge) {
      attemptBadge.textContent = `GUESS ${Math.min(currentAttempt + 1, MAX_ATTEMPTS)} / ${MAX_ATTEMPTS}`;
    }
    if (attemptProgressBar) {
      const pct = (currentAttempt / MAX_ATTEMPTS) * 100;
      attemptProgressBar.style.width = `${pct}%`;
    }
  }

  // 5. 3 Progressive Hints Logic
  function updateHintsUI() {
    document.getElementById('hint-text-1').textContent = currentHints.hint_1 || 'First letter: C';
    document.getElementById('hint-text-2').textContent = currentHints.hint_2 || 'Contains A and E';
    document.getElementById('hint-text-3').textContent = currentHints.hint_3 || 'Meaning: Bird or crane machine';
  }

  document.getElementById('unlock-hint-1').addEventListener('click', () => {
    document.getElementById('hint-body-1').classList.remove('blur-hint');
    showToast('💡 Hint 1 Unlocked!');
  });
  document.getElementById('unlock-hint-2').addEventListener('click', () => {
    document.getElementById('hint-body-2').classList.remove('blur-hint');
    showToast('💡 Hint 2 Unlocked!');
  });
  document.getElementById('unlock-hint-3').addEventListener('click', () => {
    document.getElementById('hint-body-3').classList.remove('blur-hint');
    showToast('💡 Hint 3 Unlocked!');
  });

  // 6. Reset Countdown Timer (Countdown to Midnight UTC)
  function startCountdown() {
    setInterval(() => {
      const now = new Date();
      const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1);
      const diff = tomorrow - now;

      const hrs = String(Math.floor((diff / (1000 * 60 * 60)) % 24)).padStart(2, '0');
      const mins = String(Math.floor((diff / (1000 * 60)) % 60)).padStart(2, '0');
      const secs = String(Math.floor((diff / 1000) % 60)).padStart(2, '0');

      if (countdownTimer) {
        countdownTimer.textContent = `${hrs}:${mins}:${secs}`;
      }
    }, 1000);
  }
  startCountdown();

  // 7. Global Leaderboard Fetch
  async function loadLeaderboard() {
    try {
      const res = await fetch('/leaderboard');
      const data = await res.json();
      leaderboardBody.innerHTML = '';
      if (data.leaderboard) {
        data.leaderboard.forEach(item => {
          const medal = item.rank === 1 ? '🥇' : (item.rank === 2 ? '🥈' : (item.rank === 3 ? '🥉' : `#${item.rank}`));
          const tr = document.createElement('tr');
          tr.innerHTML = `
            <td><strong>${medal}</strong></td>
            <td><strong>${item.name}</strong></td>
            <td>${item.games_won}</td>
            <td>🔥 ${item.current_streak}</td>
          `;
          leaderboardBody.appendChild(tr);
        });
      }
    } catch (err) {
      console.error(err);
    }
  }

  // 8. Friends List Fetch & Add Friend
  async function loadFriends() {
    try {
      const res = await fetch(`/friends?player=${encodeURIComponent(currentPlayer)}`);
      const data = await res.json();
      friendsList.innerHTML = '';
      if (data.friends) {
        data.friends.forEach(f => {
          const div = document.createElement('div');
          div.className = 'friend-item';
          div.innerHTML = `
            <div>
              <strong>${f.name}</strong>
              <div style="font-size:0.75rem; color:var(--text-muted);">Streak: 🔥 ${f.streak}</div>
            </div>
            <span style="font-size:0.8rem; color:var(--color-green); font-weight:600;">${f.status}</span>
          `;
          friendsList.appendChild(div);
        });
      }
    } catch (err) {
      console.error(err);
    }
  }

  addFriendBtn.addEventListener('click', async () => {
    const friend = friendNameInput.value.trim();
    if (friend) {
      await fetch('/friends', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ player: currentPlayer, friend })
      });
      showToast(`Added ${friend} to friends!`);
      friendNameInput.value = '';
      loadFriends();
    }
  });

  // 9. Auth Login / Signup Form
  authForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    const username = document.getElementById('auth-username').value.trim();
    const password = document.getElementById('auth-password').value.trim();
    if (!username) return;

    try {
      const res = await fetch('/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
      });
      const data = await res.json();
      if (res.ok) {
        showToast(`Authenticated as ${data.player}!`);
        authModal.classList.add('hidden');
        loadGameState(data.player);
      } else {
        showToast(`Auth error: ${data.error || 'Invalid credentials'}`);
      }
    } catch (err) {
      console.error(err);
      showToast('Auth server error');
    }
  });

  // 10. Admin Panel Manage Words
  async function loadAdminWords() {
    try {
      const res = await fetch('/admin/words?admin_key=admin123');
      const data = await res.json();
      adminWordsBody.innerHTML = '';
      if (data.words) {
        data.words.forEach(w => {
          const tr = document.createElement('tr');
          tr.innerHTML = `
            <td>${w.game_date}</td>
            <td><strong>${w.word}</strong></td>
            <td>${w.hint_1}</td>
            <td>${w.hint_2}</td>
            <td>${w.hint_3}</td>
          `;
          adminWordsBody.appendChild(tr);
        });
      }
    } catch (err) {
      console.error(err);
    }
  }

  adminWordForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    const game_date = document.getElementById('admin-date').value;
    const word = document.getElementById('admin-word').value.trim();
    const hint_1 = document.getElementById('admin-hint1').value.trim();
    const hint_2 = document.getElementById('admin-hint2').value.trim();
    const hint_3 = document.getElementById('admin-hint3').value.trim();

    try {
      const res = await fetch('/admin/words?admin_key=admin123', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ game_date, word, hint_1, hint_2, hint_3 })
      });
      if (res.ok) {
        showToast(`Added word ${word.toUpperCase()} with 3 hints!`);
        adminWordForm.reset();
        loadAdminWords();
      } else {
        showToast('Failed to save admin word.');
      }
    } catch (err) {
      console.error(err);
    }
  });

  // 11. Share Results Feature
  shareResultsBtn.addEventListener('click', () => {
    const dateStr = new Date().toISOString().split('T')[0];
    let shareText = `Daily Word Game ${dateStr} ${activeGameGuesses.length}/6 🎯\n\n`;

    activeGameGuesses.forEach(g => {
      const feedback = g.feedback || [];
      for (let c = 0; c < WORD_LENGTH; c++) {
        const st = parseStatus(feedback[c]);
        if (st === 'GREEN') shareText += '🟩';
        else if (st === 'YELLOW') shareText += '🟨';
        else shareText += '⬛';
      }
      shareText += '\n';
    });

    shareText += '\nPlay now: http://localhost:8080';

    if (navigator.clipboard) {
      navigator.clipboard.writeText(shareText).then(() => {
        showToast('📋 Game results copied to clipboard!');
      });
    } else {
      showToast('Result preview generated!');
    }
  });

  // 12. Victory Effects & Utilities
  function triggerVictoryEffects() {
    showToast('🏆 PERFECT! Target Word Solved!', 4000);
    if (window.confetti) {
      window.confetti({
        particleCount: 160,
        spread: 90,
        origin: { y: 0.55 },
        colors: ['#10b981', '#f59e0b', '#6366f1', '#ec4899', '#ffffff']
      });
    }
  }

  function updateStatsUI(streak, maxStreak, played, won, solved) {
    statPlayed.textContent = played || (solved ? 1 : 0);
    const pct = (played > 0) ? Math.round((won / played) * 100) : (solved ? 100 : 0);
    statWinRate.textContent = `${pct}%`;
    statStreak.textContent = streak || (solved ? 1 : 0);
    statMaxStreak.textContent = maxStreak || (solved ? 1 : 0);
    gameStatusSummary.textContent = solved 
      ? `🎉 Puzzle Solved!` 
      : (currentAttempt >= 6 ? `❌ Out of guesses today!` : `In Progress (${currentAttempt}/6 guesses used)`);
  }

  function showToast(message, duration = 2400) {
    const toast = document.createElement('div');
    toast.className = 'toast';
    toast.textContent = message;
    toastContainer.appendChild(toast);
    setTimeout(() => toast.remove(), duration + 300);
  }

  // Event Listeners for UI
  document.getElementById('keyboard').addEventListener('click', (e) => {
    const btn = e.target.closest('.key');
    if (btn) handleInputKey(btn.getAttribute('data-key'));
  });

  document.addEventListener('keydown', (e) => {
    if (e.target.tagName === 'INPUT') return;
    if (e.key === 'Enter') handleInputKey('ENTER');
    else if (e.key === 'Backspace') handleInputKey('BACKSPACE');
    else if (/^[a-zA-Z]$/.test(e.key)) handleInputKey(e.key);
  });

  switchPlayerBtn.addEventListener('click', () => {
    const name = playerInput.value.trim();
    if (name) {
      loadGameState(name);
      showToast(`Loaded Profile: ${name}`);
    }
  });

  document.querySelectorAll('.preset-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      const name = btn.getAttribute('data-player');
      loadGameState(name);
      showToast(`Switched to: ${name}`);
    });
  });

  // Modal toggle listeners
  hintsBtn.addEventListener('click', () => hintsModal.classList.remove('hidden'));
  closeHintsBtn.addEventListener('click', () => hintsModal.classList.add('hidden'));

  leaderboardBtn.addEventListener('click', () => {
    leaderboardModal.classList.remove('hidden');
    loadLeaderboard();
  });
  closeLeaderboardBtn.addEventListener('click', () => leaderboardModal.classList.add('hidden'));

  friendsBtn.addEventListener('click', () => {
    friendsModal.classList.remove('hidden');
    loadFriends();
  });
  closeFriendsBtn.addEventListener('click', () => friendsModal.classList.add('hidden'));

  authBtn.addEventListener('click', () => authModal.classList.remove('hidden'));
  closeAuthBtn.addEventListener('click', () => authModal.classList.add('hidden'));

  adminBtn.addEventListener('click', () => {
    adminModal.classList.remove('hidden');
    loadAdminWords();
  });
  closeAdminBtn.addEventListener('click', () => adminModal.classList.add('hidden'));

  statsBtn.addEventListener('click', () => statsModal.classList.remove('hidden'));
  closeStatsBtn.addEventListener('click', () => statsModal.classList.add('hidden'));

  // Start Game
  loadGameState(currentPlayer);
});
