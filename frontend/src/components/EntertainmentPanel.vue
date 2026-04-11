<template>
  <Teleport to="body">
    <Transition name="ent-slide">
      <div v-if="open" class="ent-panel">

        <div class="ent-header">
          <span class="ent-title">🎮 РАЗВЛЕЧЕНИЯ</span>
          <button class="ent-close" @click="$emit('close')" title="Закрыть">✕</button>
        </div>

        <div class="ent-list">

          <!-- ── Аниме Квиз ─────────────────────────────────────────────── -->
          <div class="game-card" :class="{ active: quizActive }">
            <div class="card-top">
              <div class="card-icon">🎌</div>
              <div class="card-body">
                <div class="card-name">Аниме Квиз</div>
                <div class="card-desc">Угадай аниме по скриншотам — кто быстрее ответит, тот победит!</div>
                <div v-if="quizActive" class="card-status">
                  <span class="status-dot playing" />
                  <span class="status-text">{{ quizStatusText }}</span>
                </div>
              </div>
              <div class="card-action">
                <button
                  v-if="quizActive"
                  class="action-btn open"
                  @click="$emit('open-quiz')"
                >Открыть</button>
                <button
                  v-else
                  class="action-btn start"
                  :disabled="animeStarting"
                  @click="launchAnimeQuiz"
                >{{ animeStarting ? '…' : 'Запустить' }}</button>
              </div>
            </div>

            <!-- Settings (only when not active) -->
            <div v-if="!quizActive" class="settings-block">
              <div class="setting-row">
                <span class="setting-label">Раундов</span>
                <div class="rounds-picker">
                  <button
                    v-for="n in ROUND_OPTIONS"
                    :key="n"
                    class="round-btn"
                    :class="{ selected: animeSettings.rounds === n }"
                    @click="animeSettings.rounds = n"
                  >{{ n }}</button>
                </div>
              </div>
            </div>
          </div>

          <!-- ── Музыкальный Квиз ───────────────────────────────────────────── -->
          <div class="game-card" :class="{ active: musicQuizActive }">
            <div class="card-top">
              <div class="card-icon">🎵</div>
              <div class="card-body">
                <div class="card-name">Музыкальный Квиз</div>
                <div class="card-desc">Угадай аниме по опенингу или эндингу. Треки — из базы AnimeThemes!</div>
                <div v-if="musicQuizActive" class="card-status">
                  <span class="status-dot playing" />
                  <span class="status-text">{{ musicQuizStatusText }}</span>
                </div>
                <!-- Pool status while idle -->
                <div v-if="!musicQuizActive" class="pool-status">
                  <template v-if="musicQuizState?.poolError">
                    <span class="pool-err">⚠ {{ musicQuizState.poolError }}</span>
                    <button class="pool-retry-btn" :disabled="poolRetrying" @click="retryPool">
                      {{ poolRetrying ? '…' : '↺ Повторить' }}
                    </button>
                  </template>
                  <template v-else-if="!musicQuizState?.poolReady">
                    <span class="pool-loading">⏳ Загрузка треков… ({{ musicQuizState?.poolCount ?? 0 }})</span>
                  </template>
                  <template v-else>
                    <span class="pool-ok">
                      ✓ {{ musicQuizState.poolCount }} треков
                      <span v-if="musicQuizState.poolPopular > 0" class="pool-popular">
                        · {{ musicQuizState.poolPopular }} из топ MAL
                      </span>
                    </span>
                  </template>
                </div>
              </div>
              <div class="card-action">
                <button
                  v-if="musicQuizActive"
                  class="action-btn open"
                  @click="$emit('open-music-quiz')"
                >Открыть</button>
                <button
                  v-else
                  class="action-btn start music"
                  :disabled="!musicQuizState?.poolReady || musicStarting"
                  @click="launchMusicQuiz"
                >{{ musicStarting ? '…' : 'Запустить' }}</button>
              </div>
            </div>

            <!-- Settings (only when not active) -->
            <div v-if="!musicQuizActive" class="settings-block">
              <div class="setting-row">
                <span class="setting-label">Раундов</span>
                <div class="rounds-picker">
                  <button
                    v-for="n in ROUND_OPTIONS"
                    :key="n"
                    class="round-btn"
                    :class="{ selected: musicSettings.rounds === n }"
                    @click="musicSettings.rounds = n"
                  >{{ n }}</button>
                </div>
              </div>
              <div class="setting-row">
                <span class="setting-label">Типы</span>
                <div class="type-picker">
                  <button
                    v-for="t in THEME_TYPES"
                    :key="t.value"
                    class="type-btn"
                    :class="{ selected: isMusicTypeSelected(t.value) }"
                    @click="toggleMusicType(t.value)"
                  >{{ t.label }}</button>
                </div>
              </div>
            </div>
          </div>

        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, reactive } from 'vue'
import { retryPoolLoad } from '../composables/useMusicQuiz.js'

const props = defineProps({
  open:           { type: Boolean, default: false },
  quizPhase:      { type: String,  default: 'idle' },
  musicQuizPhase: { type: String,  default: 'idle' },
  musicQuizState: { type: Object,  default: null },
})

const emit = defineEmits([
  'close',
  'start-quiz',  'open-quiz',
  'start-music-quiz', 'open-music-quiz',
])

// ─── Constants ────────────────────────────────────────────────────────────────
const ROUND_OPTIONS = [5, 10, 15, 20]
const THEME_TYPES   = [
  { value: 'OP', label: 'OP'  },
  { value: 'ED', label: 'ED'  },
  { value: 'IN', label: 'IN'  },
]

// ─── Local settings state ─────────────────────────────────────────────────────
const animeSettings = reactive({ rounds: 10 })
const musicSettings = reactive({ rounds: 10, allowedTypes: [] }) // empty = all

function isMusicTypeSelected(type) {
  return musicSettings.allowedTypes.length === 0 || musicSettings.allowedTypes.includes(type)
}

function toggleMusicType(type) {
  const all = THEME_TYPES.map(t => t.value)
  // If currently "all" selected, clicking one deselects the others
  if (musicSettings.allowedTypes.length === 0) {
    musicSettings.allowedTypes = all.filter(t => t !== type)
    return
  }
  const idx = musicSettings.allowedTypes.indexOf(type)
  if (idx === -1) {
    musicSettings.allowedTypes.push(type)
    if (musicSettings.allowedTypes.length === all.length) {
      musicSettings.allowedTypes = [] // back to "all"
    }
  } else {
    if (musicSettings.allowedTypes.length === 1) return // can't deselect last
    musicSettings.allowedTypes.splice(idx, 1)
  }
}

// ─── Status computed ──────────────────────────────────────────────────────────
const quizActive      = computed(() => props.quizPhase !== 'idle')
const musicQuizActive = computed(() => props.musicQuizPhase !== 'idle')

const quizStatusText = computed(() => ({
  lobby:   'Ожидание игроков…',
  playing: 'Игра идёт!',
  results: 'Результаты',
}[props.quizPhase] ?? ''))

const musicQuizStatusText = computed(() => ({
  lobby:   'Ожидание игроков…',
  playing: 'Игра идёт!',
  results: 'Результаты',
}[props.musicQuizPhase] ?? ''))

// ─── Pool retry ───────────────────────────────────────────────────────────────
const poolRetrying = ref(false)

async function retryPool() {
  poolRetrying.value = true
  try {
    await retryPoolLoad()
  } finally {
    poolRetrying.value = false
  }
}

// ─── Launch handlers ──────────────────────────────────────────────────────────
const animeStarting = ref(false)
const musicStarting = ref(false)

async function launchAnimeQuiz() {
  animeStarting.value = true
  try {
    await emit('start-quiz', { rounds: animeSettings.rounds })
  } finally {
    animeStarting.value = false
  }
}

async function launchMusicQuiz() {
  musicStarting.value = true
  try {
    await emit('start-music-quiz', {
      rounds:       musicSettings.rounds,
      allowedTypes: musicSettings.allowedTypes.length > 0 ? [...musicSettings.allowedTypes] : [],
    })
  } finally {
    musicStarting.value = false
  }
}
</script>

<style scoped>
.ent-panel {
  position: fixed;
  top: var(--header-h, 56px);
  right: 0;
  bottom: var(--bar-h, 72px);
  width: 300px;
  background: rgba(9, 9, 20, 0.97);
  border-left: 1px solid #1e1e3f;
  display: flex;
  flex-direction: column;
  z-index: 50;
  backdrop-filter: blur(12px);
}

/* ─── Slide transition ───────────────────────────────────────────────────── */
.ent-slide-enter-active,
.ent-slide-leave-active {
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
.ent-slide-enter-from,
.ent-slide-leave-to {
  transform: translateX(100%);
}

/* ─── Header ─────────────────────────────────────────────────────────────── */
.ent-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  height: 44px;
  border-bottom: 1px solid #1e1e3f;
  flex-shrink: 0;
}

.ent-title {
  font-family: 'Orbitron', sans-serif;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 2px;
  color: #c8a0f0;
}

.ent-close {
  background: none;
  border: none;
  color: #7070a0;
  font-size: 14px;
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 4px;
  transition: color 0.15s, background 0.15s;
}
.ent-close:hover { color: #ff2957; background: rgba(255,41,87,0.1); }

/* ─── List ───────────────────────────────────────────────────────────────── */
.ent-list {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  scrollbar-width: thin;
  scrollbar-color: #2e2e5f transparent;
}

/* ─── Card ───────────────────────────────────────────────────────────────── */
.game-card {
  border-radius: 12px;
  background: rgba(30, 30, 63, 0.4);
  border: 1px solid #2e2e5f;
  transition: border-color 0.2s, background 0.2s;
  overflow: hidden;
}
.game-card.active {
  background: rgba(157,78,221,0.1);
  border-color: rgba(157,78,221,0.4);
  box-shadow: 0 0 16px rgba(157,78,221,0.1);
}

.card-top {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px;
}

.card-icon {
  font-size: 28px;
  flex-shrink: 0;
  padding-top: 2px;
}

.card-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.card-name {
  font-weight: 700;
  font-size: 14px;
  color: #c8c8e8;
}

.card-desc {
  font-size: 11px;
  color: #7070a0;
  line-height: 1.5;
}

.card-status {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-top: 2px;
}

.status-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}
.status-dot.playing {
  background: #ff6b9d;
  box-shadow: 0 0 6px #ff6b9d;
  animation: blink 1.2s ease-in-out infinite;
}
@keyframes blink { 0%,100% { opacity: 1; } 50% { opacity: 0.4; } }

.status-text {
  font-size: 10px;
  color: #9d4edd;
  font-weight: 600;
}

.pool-status {
  font-size: 10px;
  margin-top: 2px;
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.pool-loading { color: #7070a0; }
.pool-ok      { color: #00c896; }
.pool-err     { color: #ff4d6d; }
.pool-popular { color: #00b4d8; opacity: .85; }
.pool-retry-btn {
  padding: 1px 7px;
  border-radius: 4px;
  border: 1px solid #ff4d6d;
  background: rgba(255,77,109,0.1);
  color: #ff4d6d;
  font-size: 10px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
  flex-shrink: 0;
}
.pool-retry-btn:hover:not(:disabled) { background: rgba(255,77,109,0.2); }
.pool-retry-btn:disabled { opacity: .5; cursor: default; }

/* ─── Card action ────────────────────────────────────────────────────────── */
.card-action {
  flex-shrink: 0;
  display: flex;
  align-items: flex-start;
  padding-top: 2px;
}

.action-btn {
  padding: 6px 12px;
  border-radius: 7px;
  border: none;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}
.action-btn.start {
  background: linear-gradient(135deg, #9d4edd, #ff6b9d);
  color: #fff;
  box-shadow: 0 2px 10px rgba(157,78,221,0.3);
}
.action-btn.start.music {
  background: linear-gradient(135deg, #00b4d8, #00f5ff);
  color: #000;
  box-shadow: 0 2px 10px rgba(0,245,255,0.25);
}
.action-btn.start:hover:not(:disabled) { transform: translateY(-1px); }
.action-btn.start:disabled { opacity: .45; cursor: default; }
.action-btn.open {
  background: rgba(157,78,221,0.15);
  border: 1px solid rgba(157,78,221,0.4);
  color: #c8a0f0;
}
.action-btn.open:hover { background: rgba(157,78,221,0.25); }

/* ─── Settings block ─────────────────────────────────────────────────────── */
.settings-block {
  border-top: 1px solid #1e1e3f;
  padding: 10px 14px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: rgba(0,0,0,0.15);
}

.setting-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.setting-label {
  font-family: 'Orbitron', sans-serif;
  font-size: 8px;
  font-weight: 700;
  letter-spacing: 1.5px;
  color: #50507a;
  text-transform: uppercase;
  width: 48px;
  flex-shrink: 0;
}

/* ─── Rounds picker ──────────────────────────────────────────────────────── */
.rounds-picker,
.type-picker {
  display: flex;
  gap: 4px;
}

.round-btn,
.type-btn {
  padding: 3px 8px;
  border-radius: 5px;
  border: 1px solid #2e2e5f;
  background: transparent;
  color: #7070a0;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.12s;
}
.round-btn:hover,
.type-btn:hover  { border-color: #9d4edd; color: #c8a0f0; }
.round-btn.selected {
  background: rgba(157,78,221,0.25);
  border-color: #9d4edd;
  color: #e0c8ff;
}
.type-btn.selected {
  background: rgba(0,245,255,0.15);
  border-color: #00f5ff;
  color: #00f5ff;
}

/* ─── Mobile ─────────────────────────────────────────────────────────────── */
@media (max-width: 640px) {
  .ent-panel {
    top: var(--header-h, 48px);
    bottom: var(--bar-h, 58px);
    width: 100vw;
    border-left: none;
    border-top: 1px solid #1e1e3f;
  }
}
</style>
