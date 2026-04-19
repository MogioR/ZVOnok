import { ref } from 'vue'

const SHIKIMORI = 'https://shikimori.io'
const QUESTIONS_PER_GAME = 10

// ─── localStorage cache ───────────────────────────────────────────────────────
const CACHE_KEY    = 'zvonok_anime_pool_v2'
const CACHE_TTL_MS = 24 * 60 * 60 * 1000

function cacheRead() {
  try {
    const raw = localStorage.getItem(CACHE_KEY)
    if (!raw) return null
    const { ts, pool } = JSON.parse(raw)
    if (!Array.isArray(pool) || pool.length === 0) return null
    if (Date.now() - ts > CACHE_TTL_MS) return null
    return pool
  } catch { return null }
}

function cacheWrite(pool) {
  try {
    localStorage.setItem(CACHE_KEY, JSON.stringify({ ts: Date.now(), pool }))
  } catch {}
}

// ─── Module-level state ───────────────────────────────────────────────────────
const animePool     = ref([])   // local pool for autocomplete + screenshot prefetch
const screenshotCache = ref({})
const loadError     = ref(null)
const isLoadingPool = ref(false)

// ─── Helpers ──────────────────────────────────────────────────────────────────
function shuffle(arr) {
  const a = [...arr]
  for (let i = a.length - 1; i > 0; i--) {
    const j = (Math.random() * (i + 1)) | 0
    ;[a[i], a[j]] = [a[j], a[i]]
  }
  return a
}

// ─── Stage image helpers ──────────────────────────────────────────────────────
export function getStageImage(animeId, poster, stageIndex) {
  if (!animeId) return poster ?? null
  const cached = screenshotCache.value[animeId]
  if (Array.isArray(cached) && cached.length > 0) {
    return cached[stageIndex % cached.length]
  }
  return poster ?? null
}

export function screenshotCount(animeId) {
  const cached = screenshotCache.value[animeId]
  return Array.isArray(cached) ? cached.length : 0
}

// ─── Shikimori API (via Go proxy) ─────────────────────────────────────────────
async function fetchPage(page) {
  const res = await fetch(`/api/quiz/animes?limit=50&page=${page}`)
  if (!res.ok) throw new Error(`Shikimori вернул ${res.status}`)
  const data = await res.json()
  if (!Array.isArray(data)) throw new Error('Неожиданный формат ответа')
  return data
}

let _fetchPromise = null

async function _fetchFromApi() {
  const allResults = []
  for (let page = 1; page <= 20; page++) {
    const result = await Promise.allSettled([fetchPage(page)])
    allResults.push(...result)
    await new Promise(r => setTimeout(r, 3000))
  }
  const raw = allResults.filter(p => p.status === 'fulfilled').flatMap(p => p.value)
  if (raw.length === 0) {
    const failed = allResults.find(p => p.status === 'rejected')
    throw new Error(failed?.reason?.message ?? 'Нет данных от Shikimori')
  }
  return raw.map(a => ({
    id:      a.id,
    name:    a.name,
    russian: a.russian ?? a.name,
    poster:  SHIKIMORI + (a.image?.original ?? `/system/animes/original/${a.id}.jpg`),
  }))
}

async function loadAnimes() {
  if (animePool.value.length >= QUESTIONS_PER_GAME) return
  const cached = cacheRead()
  if (cached) {
    animePool.value = shuffle(cached)
    return
  }
  if (!_fetchPromise) {
    _fetchPromise = _fetchFromApi().then(pool => {
      cacheWrite(pool)
      return pool
    }).finally(() => { _fetchPromise = null })
  }
  const pool = await _fetchPromise
  animePool.value = shuffle(pool)
}

export async function loadScreenshots(animeId) {
  if (screenshotCache.value[animeId]) return
  screenshotCache.value = { ...screenshotCache.value, [animeId]: 'loading' }
  try {
    const res  = await fetch(`/api/quiz/screenshots?id=${animeId}`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    const urls = Array.isArray(data)
      ? data.map(s => SHIKIMORI + s.original).filter(Boolean)
      : []
    screenshotCache.value = { ...screenshotCache.value, [animeId]: urls.length > 0 ? urls : 'error' }
  } catch {
    screenshotCache.value = { ...screenshotCache.value, [animeId]: 'error' }
  }
}

export async function prefetchScreenshots(animeIds) {
  for (const id of animeIds) {
    if (!screenshotCache.value[id]) {
      await loadScreenshots(id)
      await new Promise(r => setTimeout(r, 120))
    }
  }
}

// ─── Room awareness ───────────────────────────────────────────────────────────
const _currentRoom = { value: 'default' }
export function setQuizRoom(id) { _currentRoom.value = id || 'default' }

function quizUrl(path) { return `/api/quiz/${path}?room=${_currentRoom.value}` }

// ─── HTTP actions → server quiz API ──────────────────────────────────────────
async function _post(path, body) {
  const res = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res
}

// Open the game lobby.  Sends a randomised question list to the server.
// player:   { id, name, avatar }
// settings: { rounds: number }
export async function openLobby(player, settings = {}) {
  const rounds = settings.rounds ?? QUESTIONS_PER_GAME
  loadError.value = null
  isLoadingPool.value = true
  try {
    if (animePool.value.length < rounds) {
      await loadAnimes()
    }
    const questions = shuffle(animePool.value)
      .slice(0, Math.max(rounds, 20))  // send a bit more so server can trim to exactly rounds
      .map(({ id, name, russian, poster }) => ({ id, name, russian, poster }))

    await _post(quizUrl('lobby'), {
      settings: { rounds },
      questions,
      players: player ? [{ id: player.id, name: player.name, avatar: player.avatar ?? '' }] : [],
    })

    // Prefetch screenshots for the first few questions asynchronously
    const ids = questions.slice(0, 5).map(q => q.id)
    prefetchScreenshots(ids).catch(() => {})
  } catch (e) {
    console.error('[AnimeQuiz] openLobby:', e)
    loadError.value = e.message ?? 'Ошибка загрузки данных'
    throw e
  } finally {
    isLoadingPool.value = false
  }
}

export async function retryLoad(player, settings) {
  animePool.value       = []
  screenshotCache.value = {}
  loadError.value       = null
  await openLobby(player, settings)
}

export async function joinLobby(player) {
  await _post(quizUrl('join'), { id: player.id, name: player.name, avatar: player.avatar ?? '' })
}

export async function startGame() {
  await _post(quizUrl('start'), {})
}

export async function submitAnswer(playerId, playerName, text) {
  const res  = await _post(quizUrl('answer'), { playerId, playerName, text })
  const json = await res.json()
  return json
}

export async function stopGame() {
  await _post(quizUrl('stop'), {})
}

export async function playAgain() {
  await _post(quizUrl('again'), {})
}

// ─── Background prefetch on module load ──────────────────────────────────────
setTimeout(() => {
  if (animePool.value.length === 0) {
    loadAnimes().catch(e => console.warn('[AnimeQuiz] Background prefetch:', e))
  }
}, 5000)

// ─── Export ───────────────────────────────────────────────────────────────────
export function useAnimeQuiz() {
  return {
    animePool,
    screenshotCache,
    loadError,
    isLoadingPool,
    QUESTIONS_PER_GAME,
    getStageImage,
    screenshotCount,
    loadScreenshots,
    prefetchScreenshots,
    openLobby,
    retryLoad,
    joinLobby,
    startGame,
    submitAnswer,
    stopGame,
    playAgain,
    setQuizRoom,
  }
}
