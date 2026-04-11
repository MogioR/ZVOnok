// Music quiz composable — thin HTTP wrapper.
// All theme pool management and question selection is done server-side.

import { ref } from 'vue'

// ─── Anime names for autocomplete ────────────────────────────────────────────

const NAMES_CACHE_KEY = 'zvonok_music_animenames_v1'

export const musicThemeNames = ref([])

async function loadMusicAnimeNames() {
  try {
    const cached = localStorage.getItem(NAMES_CACHE_KEY)
    if (cached) {
      const parsed = JSON.parse(cached)
      if (Array.isArray(parsed) && parsed.length > 0) {
        musicThemeNames.value = parsed
        return
      }
    }
  } catch {}
  try {
    const res = await fetch('/api/musicquiz/animenames')
    if (res.ok) {
      const data = await res.json()
      if (Array.isArray(data) && data.length > 0) {
        musicThemeNames.value = data
        try { localStorage.setItem(NAMES_CACHE_KEY, JSON.stringify(data)) } catch {}
      }
    }
  } catch {}
}

// Load after a delay — pool may not be ready immediately on server start
setTimeout(loadMusicAnimeNames, 10000)

// ─── HTTP helper ──────────────────────────────────────────────────────────────
async function _post(path, body) {
  const res = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const text = await res.text().catch(() => `HTTP ${res.status}`)
    throw new Error(text || `HTTP ${res.status}`)
  }
  return res
}

// ─── Actions → server musicquiz API ──────────────────────────────────────────

// Open the music quiz lobby.
// player:   { id, name, avatar }
// settings: { rounds: number, allowedTypes: string[] }
export async function openLobby(player, settings = {}) {
  await _post('/api/musicquiz/lobby', {
    settings: {
      rounds:       settings.rounds       ?? 10,
      allowedTypes: settings.allowedTypes ?? [],
    },
    players: player ? [{ id: player.id, name: player.name, avatar: player.avatar ?? '' }] : [],
  })
}

export async function joinLobby(player) {
  await _post('/api/musicquiz/join', { id: player.id, name: player.name, avatar: player.avatar ?? '' })
}

export async function startGame() {
  await _post('/api/musicquiz/start', {})
}

export async function submitAnswer(playerId, playerName, text) {
  const res  = await _post('/api/musicquiz/answer', { playerId, playerName, text })
  const json = await res.json()
  return json  // { correct: bool }
}

export async function stopGame() {
  await _post('/api/musicquiz/stop', {})
}

export async function playAgain() {
  await _post('/api/musicquiz/again', {})
}

// ─── Export ───────────────────────────────────────────────────────────────────
export function useMusicQuiz() {
  return {
    openLobby,
    joinLobby,
    startGame,
    submitAnswer,
    stopGame,
    playAgain,
  }
}
