<template>
  <Teleport to="body">
    <Transition name="music-modal">
      <div v-if="open" class="music-overlay" @click.self="$emit('close')">
        <div class="music-panel">

          <!-- ─── Header ──────────────────────────────────────────────────── -->
          <div class="panel-header">
            <div class="panel-title">
              <span class="bot-icon">🎵</span>
              МУЗЫКАНТ БОТ
            </div>
            <button class="close-btn" @click="$emit('close')">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
              </svg>
            </button>
          </div>

          <!-- ─── Search ──────────────────────────────────────────────────── -->
          <div class="search-section">
            <div class="search-bar">
              <svg class="search-icon" width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                <path d="M15.5 14h-.79l-.28-.27A6.471 6.471 0 0 0 16 9.5 6.5 6.5 0 1 0 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/>
              </svg>
              <input
                ref="searchInputEl"
                v-model="query"
                class="search-input"
                placeholder="Поиск, видео или плейлист YouTube…"
                @keydown.enter="handleSearch"
                @input="onQueryInput"
              />
              <button v-if="query" class="search-clear" @click="query = ''; searchResults = []">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
                </svg>
              </button>
              <button
                class="search-btn"
                :class="{ loading: searching }"
                :disabled="searching || !query.trim()"
                @click="handleSearch"
              >
                <svg v-if="!searching" width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M8 5v14l11-7z"/>
                </svg>
                <svg v-else class="spin" width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6s-2.69 6-6 6-6-2.69-6-6H4c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8z"/>
                </svg>
              </button>
            </div>

            <div v-if="playlistHintVisible" class="playlist-hint">
              <span class="playlist-hint-text">Обнаружен плейлист — можно добавить все ролики в очередь.</span>
              <button
                class="playlist-add-all"
                type="button"
                :disabled="playlistAdding"
                @click="addPlaylistFromURL(query.trim())"
              >
                {{ playlistAdding ? 'Загрузка…' : 'Весь плейлист в очередь' }}
              </button>
            </div>
            <p v-if="playlistToast" class="playlist-toast">{{ playlistToast }}</p>

            <!-- Search results -->
            <Transition name="results">
              <div v-if="searchResults.length" class="search-results">
                <div
                  v-for="r in searchResults"
                  :key="r.url"
                  class="result-item"
                  @click="addResult(r)"
                >
                  <img
                    v-if="r.thumbnail"
                    :src="r.thumbnail"
                    class="result-thumb"
                    alt=""
                    @error="$event.target.style.display='none'"
                  />
                  <div class="result-thumb-placeholder" v-else>♪</div>
                  <div class="result-info">
                    <div class="result-title">{{ r.title }}</div>
                    <div v-if="r.duration" class="result-dur">{{ fmtDuration(r.duration) }}</div>
                  </div>
                  <button class="result-add">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
                    </svg>
                  </button>
                </div>
              </div>
            </Transition>
          </div>

          <!-- ─── Progress ────────────────────────────────────────────────── -->
          <div class="progress-row">
            <span class="prog-time">{{ fmtDuration(Math.floor(position)) }}</span>
            <div class="prog-bar-wrap">
              <div class="prog-bar-fill" :style="{ width: progressPct + '%' }"></div>
            </div>
            <span class="prog-time">{{ state.current?.duration ? fmtDuration(state.current.duration) : '--:--' }}</span>
          </div>

          <!-- ─── Now playing ─────────────────────────────────────────────── -->
          <div v-if="state.current" class="now-playing">
            <div class="now-label">СЕЙЧАС ИГРАЕТ</div>
            <div class="now-track">
              <div class="now-thumb-wrap">
                <img
                  v-if="state.current.thumbnail"
                  :src="state.current.thumbnail"
                  class="now-thumb"
                  alt=""
                  @error="$event.target.style.display='none'"
                />
                <div v-else class="now-thumb-ph">♪</div>
                <div class="now-bars">
                  <span /><span /><span /><span />
                </div>
              </div>
              <div class="now-info">
                <div class="now-title">{{ state.current.title }}</div>
                <div v-if="state.current.duration" class="now-dur">
                  {{ fmtDuration(state.current.duration) }}
                </div>
              </div>
              <button class="skip-btn" title="Следующий трек" @click="skip">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M6 18l8.5-6L6 6v12zm2-8.14L11.03 12 8 14.14V9.86zM16 6h2v12h-2z"/>
                </svg>
              </button>
            </div>
          </div>

          <!-- ─── Queue ────────────────────────────────────────────────────── -->
          <div class="queue-section">
            <div class="queue-header">
              <span class="queue-label">
                ОЧЕРЕДЬ
                <span v-if="state.queue.length" class="queue-count">{{ state.queue.length }}</span>
              </span>
              <button
                v-if="state.queue.length"
                class="clear-btn"
                title="Очистить очередь"
                @click="clear"
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
                </svg>
                Очистить
              </button>
            </div>

            <div v-if="!state.queue.length && !state.current" class="queue-empty">
              <svg width="32" height="32" viewBox="0 0 24 24" fill="currentColor" style="opacity:.25">
                <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
              </svg>
              <span>Очередь пуста</span>
            </div>

            <div v-else class="queue-list">
              <TransitionGroup name="queue-item">
                <div
                  v-for="(track, i) in state.queue"
                  :key="track.url + i"
                  class="queue-row"
                >
                  <span class="queue-num">{{ i + 1 }}</span>
                  <img
                    v-if="track.thumbnail"
                    :src="track.thumbnail"
                    class="queue-thumb"
                    alt=""
                    @error="$event.target.style.display='none'"
                  />
                  <div class="queue-info">
                    <div class="queue-title">{{ track.title }}</div>
                    <div v-if="track.duration" class="queue-dur">{{ fmtDuration(track.duration) }}</div>
                  </div>
                  <button class="queue-del" title="Удалить" @click="remove(i)">
                    <svg width="13" height="13" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zm2.46-7.12 1.41-1.41L12 12.59l2.12-2.12 1.41 1.41L13.41 14l2.12 2.12-1.41 1.41L12 15.41l-2.12 2.12-1.41-1.41L10.59 14l-2.13-2.12z"/>
                    </svg>
                  </button>
                </div>
              </TransitionGroup>
            </div>
          </div>

        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, watch, nextTick, computed, onUnmounted } from 'vue'

const props = defineProps({
  open:   { type: Boolean, required: true },
  state:  { type: Object,  required: true },
  roomId: { type: String,  default: 'default' },
})
const emit = defineEmits(['close'])

function apiUrl(path) {
  return `/api/music/${path}?room=${props.roomId}`
}

// ─── Playback progress ────────────────────────────────────────────────────────
const position = ref(0)
let posTimer = null

watch(
  () => props.state?.startedAt,
  (startedAt) => {
    if (posTimer) { clearInterval(posTimer); posTimer = null }
    if (!startedAt) { position.value = 0; return }
    const tick = () => { position.value = (Date.now() - startedAt) / 1000 }
    tick()
    posTimer = setInterval(tick, 1000)
  },
  { immediate: true },
)

onUnmounted(() => {
  if (posTimer) clearInterval(posTimer)
  if (playlistToastTimer) clearTimeout(playlistToastTimer)
})

const progressPct = computed(() => {
  const dur = props.state?.current?.duration
  if (!dur || position.value <= 0) return 0
  return Math.min(100, (position.value / dur) * 100)
})

const query        = ref('')
const searchResults = ref([])
const searching    = ref(false)
const searchInputEl = ref(null)
const playlistAdding = ref(false)
const playlistToast = ref('')
let playlistToastTimer = null

// ─── Focus search on open ──────────────────────────────────────────────────
watch(() => props.open, (v) => {
  if (v) nextTick(() => searchInputEl.value?.focus())
  else   searchResults.value = []
})

// ─── Detect YouTube URL vs search query ───────────────────────────────────
const YT_RE = /^https?:\/\/(www\.)?(youtube\.com|youtu\.be)\//
const YT_PLAYLIST_RE = /[?&]list=[^&#\s]+/i
const YT_VIDEO_ID_RE = /[?&]v=[^&#\s]+/

/**
 * Returns true only for URLs that point to an actual playlist (not a video
 * that happens to be part of a playlist).
 * - youtu.be/ID?list=...  → individual video from playlist → false
 * - youtube.com/watch?v=X&list=Y → individual video from playlist → false
 * - youtube.com/playlist?list=Y → real playlist → true
 * - youtube.com/watch?list=Y (no v=) → real playlist → true
 */
function isYtPlaylist(url) {
  if (!YT_RE.test(url)) return false
  if (!YT_PLAYLIST_RE.test(url)) return false
  // Has a specific video ID → it's a video, not the playlist itself
  if (YT_VIDEO_ID_RE.test(url)) return false
  // youtu.be/ID always refers to a specific video
  if (/youtu\.be\/[^/?]+/.test(url)) return false
  return true
}

const playlistHintVisible = computed(() => isYtPlaylist(query.value.trim()))

let debounceTimer = null
function onQueryInput() {
  clearTimeout(debounceTimer)
  if (YT_RE.test(query.value.trim()) || isYtPlaylist(query.value.trim())) {
    searchResults.value = []
    return
  }
  if (query.value.trim().length < 2) {
    searchResults.value = []
    return
  }
  // Debounce search-as-you-type for text queries
  debounceTimer = setTimeout(handleSearch, 700)
}

function showPlaylistToast(text) {
  playlistToast.value = text
  if (playlistToastTimer) clearTimeout(playlistToastTimer)
  playlistToastTimer = setTimeout(() => { playlistToast.value = '' }, 4500)
}

async function addPlaylistFromURL(url) {
  playlistAdding.value = true
  try {
    const res = await fetch(apiUrl('add-playlist'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url }),
    })
    if (!res.ok) throw new Error(String(res.status))
    const data = await res.json()
    const n = data.added ?? 0
    const cap = data.capped ? ' (достигнут лимит очереди на сервере)' : ''
    showPlaylistToast(`В очередь добавлено треков: ${n}${cap}`)
    query.value = ''
  } catch (e) {
    console.error('[music] add-playlist:', e)
    showPlaylistToast('Не удалось загрузить плейлист')
  } finally {
    playlistAdding.value = false
  }
}

async function handleSearch() {
  clearTimeout(debounceTimer)
  const q = query.value.trim()
  if (!q) return

  // Actual playlist URL (no specific video ID)
  if (isYtPlaylist(q)) {
    await addPlaylistFromURL(q)
    return
  }

  // Individual YouTube video (including videos with list= context)
  if (YT_RE.test(q)) {
    await addByURL(q, '')
    query.value = ''
    return
  }

  searching.value = true
  try {
    const res = await fetch(apiUrl('search') + `&q=${encodeURIComponent(q)}`)
    if (res.ok) searchResults.value = (await res.json()) ?? []
  } catch (e) {
    console.error('[music] search error:', e)
  } finally {
    searching.value = false
  }
}

async function addResult(track) {
  searchResults.value = []
  query.value = ''
  await addTrackToQueue(track)
}

async function addByURL(url, title) {
  await addTrackToQueue({ url, title, thumbnail: '', duration: 0 })
}

async function addTrackToQueue(track) {
  try {
    await fetch(apiUrl('add'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(track),
    })
  } catch (e) {
    console.error('[music] add error:', e)
  }
}

async function skip() {
  await fetch(apiUrl('skip'), { method: 'POST' })
}

async function remove(idx) {
  await fetch(apiUrl('remove'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ index: idx }),
  })
}

async function clear() {
  await fetch(apiUrl('clear'), { method: 'POST' })
}

function fmtDuration(sec) {
  if (!sec) return ''
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${m}:${String(s).padStart(2, '0')}`
}
</script>

<style scoped>
/* ─── Overlay ────────────────────────────────────────────────────────────── */
.music-overlay {
  position: fixed;
  inset: 0;
  z-index: 600;
  background: rgba(5, 5, 16, 0.75);
  backdrop-filter: blur(6px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.music-modal-enter-active { transition: opacity 0.2s, transform 0.2s; }
.music-modal-leave-active { transition: opacity 0.15s, transform 0.15s; }
.music-modal-enter-from, .music-modal-leave-to { opacity: 0; transform: scale(0.95); }

/* ─── Panel ──────────────────────────────────────────────────────────────── */
.music-panel {
  width: min(520px, 100%);
  max-height: 88vh;
  background: #0a0a1e;
  border: 1px solid #2e2e5f;
  border-radius: 16px;
  box-shadow: 0 24px 70px rgba(0,0,0,0.8), 0 0 0 1px rgba(157,78,221,0.1);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* ─── Header ─────────────────────────────────────────────────────────────── */
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid #1e1e3f;
  flex-shrink: 0;
  background: rgba(157, 78, 221, 0.05);
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 9px;
  font-family: 'Orbitron', sans-serif;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 2.5px;
  color: #9d4edd;
  text-shadow: 0 0 10px rgba(157,78,221,0.5);
}

.bot-icon { font-size: 18px; line-height: 1; }

.close-btn {
  display: flex; align-items: center; justify-content: center;
  width: 30px; height: 30px;
  border: 1px solid #2e2e5f; border-radius: 6px;
  background: transparent; color: #7070a0; cursor: pointer; transition: all .15s;
}
.close-btn:hover { border-color: #ff2957; color: #ff2957; }

/* ─── Progress ───────────────────────────────────────────────────────────── */
.progress-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-bottom: 1px solid #1e1e3f;
  flex-shrink: 0;
}

.prog-time {
  font-family: 'Orbitron', sans-serif;
  font-size: 9px;
  color: #7070a0;
  width: 34px;
  flex-shrink: 0;
  text-align: center;
}

.prog-bar-wrap {
  flex: 1;
  height: 3px;
  background: #1e1e3f;
  border-radius: 2px;
  overflow: hidden;
}

.prog-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #9d4edd, #00f5ff);
  border-radius: 2px;
  transition: width 0.8s linear;
}

/* ─── Search ─────────────────────────────────────────────────────────────── */
.search-section {
  padding: 12px 14px;
  border-bottom: 1px solid #1e1e3f;
  flex-shrink: 0;
}

.search-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(30, 30, 63, 0.6);
  border: 1px solid #2e2e5f;
  border-radius: 8px;
  padding: 0 8px;
}

.search-icon { color: #50507a; flex-shrink: 0; }

.search-input {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  color: #c8c8e8;
  font-size: 13px;
  padding: 9px 4px;
  font-family: inherit;
}
.search-input::placeholder { color: #50507a; }

.search-clear, .search-btn {
  display: flex; align-items: center; justify-content: center;
  width: 26px; height: 26px;
  border: none; border-radius: 5px;
  cursor: pointer; transition: all .15s; flex-shrink: 0;
}
.search-clear { background: transparent; color: #50507a; }
.search-clear:hover { color: #c8c8e8; }

.playlist-hint {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  margin-top: 10px;
  padding: 10px 12px;
  background: rgba(0, 245, 255, 0.06);
  border: 1px solid rgba(0, 245, 255, 0.25);
  border-radius: 8px;
}
.playlist-hint-text {
  font-size: 12px;
  color: #a8c8d8;
  line-height: 1.35;
  flex: 1;
  min-width: 140px;
}
.playlist-add-all {
  flex-shrink: 0;
  padding: 7px 14px;
  border-radius: 6px;
  border: 1px solid #00f5ff;
  background: rgba(0, 245, 255, 0.12);
  color: #00f5ff;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}
.playlist-add-all:hover:not(:disabled) {
  background: rgba(0, 245, 255, 0.22);
}
.playlist-add-all:disabled {
  opacity: 0.5;
  cursor: default;
}
.playlist-toast {
  margin: 8px 0 0;
  font-size: 12px;
  color: #39ff14;
}

.search-btn { background: rgba(157,78,221,0.2); color: #9d4edd; }
.search-btn:hover:not(:disabled) { background: rgba(157,78,221,0.35); color: #c8a0f0; }
.search-btn:disabled { opacity: .4; cursor: default; }
.search-btn.loading { background: rgba(0,245,255,0.1); color: #00f5ff; }

@keyframes spin { to { transform: rotate(360deg); } }
.spin { animation: spin 0.8s linear infinite; }

/* ─── Search results ─────────────────────────────────────────────────────── */
.results-enter-active, .results-leave-active { transition: opacity .15s, transform .15s; }
.results-enter-from, .results-leave-to { opacity: 0; transform: translateY(-6px); }

.search-results {
  margin-top: 8px;
  background: rgba(15,15,30,0.9);
  border: 1px solid #2e2e5f;
  border-radius: 8px;
  overflow: hidden;
  max-height: 200px;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: #2e2e5f transparent;
}

.result-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  cursor: pointer;
  transition: background .12s;
  border-bottom: 1px solid #1e1e3f;
}
.result-item:last-child { border-bottom: none; }
.result-item:hover { background: rgba(157,78,221,0.08); }

.result-thumb {
  width: 48px; height: 36px;
  object-fit: cover;
  border-radius: 4px;
  flex-shrink: 0;
}
.result-thumb-placeholder {
  width: 48px; height: 36px;
  background: #1e1e3f;
  border-radius: 4px;
  display: flex; align-items: center; justify-content: center;
  font-size: 16px; color: #50507a; flex-shrink: 0;
}

.result-info { flex: 1; min-width: 0; }
.result-title { font-size: 12px; font-weight: 600; color: #c8c8e8; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.result-dur  { font-size: 10px; color: #50507a; margin-top: 2px; }

.result-add {
  display: flex; align-items: center; justify-content: center;
  width: 24px; height: 24px;
  border: 1px solid #2e2e5f; border-radius: 4px;
  background: transparent; color: #9d4edd; cursor: pointer;
  flex-shrink: 0; transition: all .12s;
}
.result-add:hover { background: rgba(157,78,221,0.15); border-color: #9d4edd; }

/* ─── Now playing ────────────────────────────────────────────────────────── */
.now-playing {
  padding: 12px 14px 8px;
  flex-shrink: 0;
  border-bottom: 1px solid #1e1e3f;
}

.now-label {
  font-family: 'Orbitron', sans-serif;
  font-size: 8px;
  font-weight: 700;
  letter-spacing: 2px;
  color: #00f5ff;
  margin-bottom: 8px;
  opacity: .7;
}

.now-track {
  display: flex;
  align-items: center;
  gap: 10px;
}

.now-thumb-wrap {
  position: relative;
  flex-shrink: 0;
  width: 52px; height: 52px;
}

.now-thumb {
  width: 52px; height: 52px;
  object-fit: cover;
  border-radius: 6px;
}
.now-thumb-ph {
  width: 52px; height: 52px;
  background: #1e1e3f;
  border-radius: 6px;
  display: flex; align-items: center; justify-content: center;
  font-size: 22px;
}

/* Animated bars overlay */
.now-bars {
  position: absolute;
  inset: 0;
  border-radius: 6px;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  gap: 3px;
  padding: 8px 10px 6px;
  background: rgba(0,0,0,0.45);
}
.now-bars span {
  display: block;
  width: 4px;
  background: #00f5ff;
  border-radius: 2px;
  animation: bar-bounce 0.7s ease-in-out infinite alternate;
}
.now-bars span:nth-child(1) { animation-delay: 0s;    height: 60%; }
.now-bars span:nth-child(2) { animation-delay: 0.15s; height: 90%; }
.now-bars span:nth-child(3) { animation-delay: 0.3s;  height: 50%; }
.now-bars span:nth-child(4) { animation-delay: 0.45s; height: 75%; }

@keyframes bar-bounce {
  from { transform: scaleY(0.3); }
  to   { transform: scaleY(1); }
}

.now-info { flex: 1; min-width: 0; }
.now-title { font-size: 13px; font-weight: 700; color: #e8e8ff; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.now-dur   { font-size: 11px; color: #7070a0; margin-top: 3px; }

.skip-btn {
  display: flex; align-items: center; justify-content: center;
  width: 34px; height: 34px;
  border: 1px solid #2e2e5f; border-radius: 8px;
  background: rgba(0,245,255,0.05); color: #7070a0;
  cursor: pointer; transition: all .15s; flex-shrink: 0;
}
.skip-btn:hover { border-color: #00f5ff; color: #00f5ff; background: rgba(0,245,255,0.1); }

/* ─── Queue ──────────────────────────────────────────────────────────────── */
.queue-section {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.queue-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px 6px;
  flex-shrink: 0;
}

.queue-label {
  font-family: 'Orbitron', sans-serif;
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 2px;
  color: #7070a0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.queue-count {
  background: rgba(157,78,221,0.2);
  border: 1px solid #9d4edd;
  color: #c8a0f0;
  border-radius: 10px;
  padding: 1px 7px;
  font-size: 9px;
  font-family: 'Orbitron', sans-serif;
}

.clear-btn {
  display: flex; align-items: center; gap: 5px;
  padding: 4px 9px;
  border: 1px solid #2e2e5f; border-radius: 5px;
  background: transparent; color: #7070a0;
  font-size: 11px; font-weight: 600;
  cursor: pointer; transition: all .15s;
}
.clear-btn:hover { border-color: #ff2957; color: #ff2957; background: rgba(255,41,87,0.07); }

.queue-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #50507a;
  font-size: 13px;
}

.queue-list {
  flex: 1;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: #2e2e5f transparent;
  padding: 4px 0 8px;
}
.queue-list::-webkit-scrollbar { width: 4px; }
.queue-list::-webkit-scrollbar-thumb { background: #2e2e5f; border-radius: 2px; }

.queue-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 14px;
  transition: background .12s;
}
.queue-row:hover { background: rgba(157,78,221,0.06); }

.queue-item-enter-active, .queue-item-leave-active { transition: all .2s; }
.queue-item-enter-from { opacity: 0; transform: translateX(-10px); }
.queue-item-leave-to  { opacity: 0; transform: translateX(10px); }

.queue-num {
  font-family: 'Orbitron', sans-serif;
  font-size: 9px;
  color: #50507a;
  width: 14px;
  text-align: center;
  flex-shrink: 0;
}

.queue-thumb {
  width: 40px; height: 30px;
  object-fit: cover;
  border-radius: 3px;
  flex-shrink: 0;
}

.queue-info { flex: 1; min-width: 0; }
.queue-title { font-size: 12px; font-weight: 600; color: #c8c8e8; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.queue-dur   { font-size: 10px; color: #50507a; margin-top: 2px; }

.queue-del {
  display: flex; align-items: center; justify-content: center;
  width: 24px; height: 24px;
  border: none; border-radius: 4px;
  background: transparent; color: #50507a;
  cursor: pointer; transition: all .12s; flex-shrink: 0;
}
.queue-del:hover { background: rgba(255,41,87,0.12); color: #ff2957; }
</style>
