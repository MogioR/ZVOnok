<template>
  <div class="quiz-view">

    <!-- ─── Header ──────────────────────────────────────────────────────────── -->
    <div class="quiz-header">
      <div class="quiz-title">
        <span class="quiz-emoji">🎌</span>
        АНИМЕ КВИЗ
        <span class="phase-badge" :class="phase">{{ phaseLabel }}</span>
      </div>

      <div v-if="phase === 'playing'" class="quiz-progress">
        <span class="qnum">{{ currentIdx + 1 }} / {{ totalQ }}</span>
        <span class="rounds-left">осталось {{ totalQ - currentIdx - 1 }}</span>
        <div class="timer-track">
          <div class="timer-fill" :style="{ width: timerFillPct + '%' }" />
        </div>
        <span class="timer-text">{{ totalTimeLeft }}с</span>
      </div>

      <button class="stop-btn" title="Остановить игру" @click="$emit('stop')">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
          <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
        </svg>
      </button>
    </div>

    <!-- ═══════════════════════════════════════════════════════════════════════ -->
    <!-- LOADING                                                                -->
    <!-- ═══════════════════════════════════════════════════════════════════════ -->
    <div v-if="phase === 'loading'" class="center-content">
      <div class="loading-card">
        <div class="spinner" />
        <div class="loading-text">Загружаем аниме из Shikimori…</div>
        <div class="loading-sub">топ-1000 по рейтингу</div>
      </div>
    </div>

    <!-- ═══════════════════════════════════════════════════════════════════════ -->
    <!-- ERROR                                                                  -->
    <!-- ═══════════════════════════════════════════════════════════════════════ -->
    <div v-else-if="loadError" class="center-content">
      <div class="error-card">
        <div class="error-icon">⚠️</div>
        <div class="error-heading">Не удалось загрузить данные</div>
        <div class="error-msg">{{ loadError }}</div>
        <button class="retry-btn" @click="$emit('retry')">↺ Попробовать ещё раз</button>
        <button class="stop-text-btn" @click="$emit('stop')">Закрыть</button>
      </div>
    </div>

    <!-- ═══════════════════════════════════════════════════════════════════════ -->
    <!-- LOBBY                                                                  -->
    <!-- ═══════════════════════════════════════════════════════════════════════ -->
    <div v-else-if="phase === 'lobby'" class="lobby-content">
      <div class="lobby-card">
        <div class="lobby-icon">🎌</div>
        <div class="lobby-heading">Угадай аниме по кадрам!</div>
        <div class="lobby-sub">
          Каждый раунд — новое аниме из топ-1000 Shikimori.<br/>
          Угадавший первым получает <strong>3 очка</strong>, следующие — по <strong>1 очку</strong>.
        </div>

        <div v-if="players.length > 0" class="player-list">
          <div class="player-list-title">Участники ({{ players.length }})</div>
          <div class="player-row" v-for="p in players" :key="p.id">
            <div class="player-avatar-sm">
              <img v-if="p.avatar" :src="p.avatar" alt="" @error="e => e.target.style.display='none'" />
              <span v-else>{{ (p.name || '?')[0].toUpperCase() }}</span>
            </div>
            <span class="player-name-sm">{{ p.name }}</span>
            <span v-if="p.id === myId" class="you-badge">вы</span>
          </div>
        </div>

        <div class="lobby-actions">
          <button v-if="!isJoined" class="join-btn" @click="$emit('join')">✋ Участвовать</button>
          <span v-else class="joined-badge">✓ Вы в игре</span>
          <button class="start-btn" :disabled="players.length === 0" @click="$emit('start')">
            ▶ Начать раунд
          </button>
        </div>

        <div class="lobby-hint">
          Другие участники могут написать <strong>+</strong> в чат, чтобы присоединиться
        </div>
      </div>
    </div>

    <!-- ═══════════════════════════════════════════════════════════════════════ -->
    <!-- PLAYING                                                                -->
    <!-- ═══════════════════════════════════════════════════════════════════════ -->
    <div v-else-if="phase === 'playing'" class="game-content">

      <!-- Image area ──────────────────────────────────────────────────────── -->
      <div class="image-wrap">
        <template v-if="animeId">
          <Transition name="img-swap" mode="out-in">
            <img
              v-if="mainImageUrl && !imgError"
              :key="imgKey"
              :src="mainImageUrl"
              class="anime-img"
              alt="Угадай аниме"
              @error="imgError = true"
            />
          </Transition>

          <div v-if="imgError" class="img-fallback">
            <span>🎌</span>
            <span>Изображение недоступно</span>
          </div>
        </template>

        <!-- Nav: previous screenshot ─────────────────────────────────────── -->
        <button
          v-if="canGoBack && !revealAnime"
          class="nav-btn nav-left"
          title="Предыдущий кадр"
          @click="goBack"
        >‹</button>

        <!-- Nav: next screenshot ─────────────────────────────────────────── -->
        <button
          v-if="canGoForward && !revealAnime"
          class="nav-btn nav-right"
          title="Следующий кадр"
          @click="goForward"
        >›</button>

        <!-- Slide label ──────────────────────────────────────────────────── -->
        <div v-if="!revealAnime" class="slide-label">
          <template v-if="viewMode === 'poster'">📷 Обложка</template>
          <template v-else>📸 Кадр {{ screenshotIdx + 1 }} / {{ availableSlides }}</template>
        </div>

        <!-- Stage dots ───────────────────────────────────────────────────── -->
        <div class="stage-dots">
          <span
            v-for="i in 4"
            :key="i"
            class="stage-dot"
            :class="{ active: stage >= i - 1, current: stage === i - 1 }"
          />
        </div>

        <!-- Reveal overlay (time ran out) ───────────────────────────────── -->
        <Transition name="reveal-fade">
          <div v-if="revealAnime" class="reveal-overlay">
            <img
              v-if="revealAnime.poster"
              :src="revealAnime.poster"
              class="reveal-poster-thumb"
              alt=""
              @error="e => e.target.style.display='none'"
            />
            <div class="reveal-label">⏰ ВРЕМЯ ВЫШЛО</div>
            <div class="reveal-name">{{ revealAnime.russian || revealAnime.name }}</div>
            <div v-if="revealAnime.russian !== revealAnime.name" class="reveal-name-en">
              {{ revealAnime.name }}
            </div>
            <!-- Who scored this round -->
            <div class="reveal-scored">
              <div v-if="roundScored.length === 0" class="reveal-nobody">Никто не угадал 😔</div>
              <div
                v-for="(s, i) in roundScored"
                :key="s.playerId"
                class="reveal-scorer"
                :class="{ first: s.isFirst }"
              >
                <span class="reveal-scorer-medal">{{ i === 0 ? '🏆' : '👏' }}</span>
                <span class="reveal-scorer-name">{{ s.playerName }}</span>
                <span class="reveal-scorer-pts">+{{ s.points }} очк{{ s.points === 1 ? 'о' : 'а' }}</span>
              </div>
            </div>
          </div>
        </Transition>

        <!-- Winner flash ─────────────────────────────────────────────────── -->
        <Transition name="winner-pop">
          <div v-if="roundWinner && !revealAnime" class="winner-overlay">
            <div class="winner-icon">✓</div>
            <div class="winner-text">{{ winnerName }} угадал{{ winnerGenderSuffix }}!</div>
            <div class="winner-pts">+3 очка</div>
          </div>
        </Transition>

        <!-- Score strip (top-left) ────────────────────────────────────────── -->
        <div v-if="players.length > 0" class="score-strip">
          <div
            v-for="p in sortedScores"
            :key="p.id"
            class="score-row"
            :class="{ answered: roundAnswered[p.id] }"
          >
            <div class="score-avatar">
              <img v-if="p.avatar" :src="p.avatar" alt="" @error="e => e.target.style.display='none'" />
              <span v-else>{{ (p.name || '?')[0].toUpperCase() }}</span>
            </div>
            <span class="score-name">{{ p.name }}</span>
            <span class="score-pts">{{ p.points }}</span>
            <span v-if="roundAnswered[p.id]" class="score-check">✓</span>
          </div>
        </div>
      </div>

      <!-- Masked title hint ────────────────────────────────────────────────── -->
      <div class="hint-bar" v-if="animeId && !revealAnime">
        <span class="hint-label-sm">ПОДСКАЗКА</span>
        <span class="hint-masked">{{ maskedTitle }}</span>
      </div>

      <!-- Answer bar ──────────────────────────────────────────────────────── -->
      <div class="answer-bar" v-if="!revealAnime">

        <!-- Autocomplete suggestions -->
        <div v-if="suggestions.length > 0" class="suggestions-dropdown">
          <div
            v-for="(anime, i) in suggestions"
            :key="anime.id"
            class="suggestion-item"
            :class="{ selected: i === selectedSugIdx }"
            @mousedown.prevent="selectSuggestion(anime)"
          >
            <span class="sug-russian">{{ anime.russian }}</span>
            <span v-if="anime.name !== anime.russian" class="sug-name">{{ anime.name }}</span>
          </div>
        </div>

        <template v-if="isAnswered">
          <div class="answered-badge">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
              <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
            </svg>
            Правильно! Жди следующего вопроса…
          </div>
        </template>

        <template v-else>
          <div class="input-wrap">
            <input
              ref="answerInputEl"
              v-model="answerDraft"
              class="answer-input"
              :class="{ shake: shaking }"
              type="text"
              placeholder="Введи название аниме…"
              autocomplete="off"
              spellcheck="false"
              maxlength="120"
              @keydown="onKeyDown"
              @blur="selectedSugIdx = -1"
            />
          </div>
          <button
            class="answer-submit"
            :disabled="!answerDraft.trim()"
            @click="submitDraft"
            title="Ответить"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
              <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/>
            </svg>
          </button>
        </template>
      </div>
    </div>

    <!-- ═══════════════════════════════════════════════════════════════════════ -->
    <!-- RESULTS                                                                -->
    <!-- ═══════════════════════════════════════════════════════════════════════ -->
    <div v-else-if="phase === 'results'" class="results-content">
      <div class="results-card">
        <div class="results-title">🏆 РЕЗУЛЬТАТЫ РАУНДА</div>
        <div class="results-list">
          <div
            v-for="(p, i) in sortedScores"
            :key="p.id"
            class="result-row"
            :class="{ winner: i === 0 }"
          >
            <span class="result-rank">{{ rankEmoji(i) }}</span>
            <div class="result-avatar">
              <img v-if="p.avatar" :src="p.avatar" alt="" @error="e => e.target.style.display='none'" />
              <span v-else>{{ (p.name || '?')[0].toUpperCase() }}</span>
            </div>
            <span class="result-name">{{ p.name }}</span>
            <span class="result-pts">{{ p.points }} очков</span>
          </div>
          <div v-if="sortedScores.length === 0" class="no-scores">
            Никто не угадал ни одного аниме 😢
          </div>
        </div>
        <div class="results-actions">
          <button class="again-btn" @click="$emit('again')">🔄 Ещё раунд</button>
          <button class="close-btn" @click="$emit('stop')">Закрыть</button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { useAnimeQuiz, getStageImage, loadScreenshots, submitAnswer } from '../composables/useAnimeQuiz.js'

const STAGE_DURATIONS = [12, 10, 8, 8]
const MAX_TIME = STAGE_DURATIONS.reduce((a, b) => a + b, 0)

const props = defineProps({
  quizState: { type: Object, default: null },
  myId:      { type: String, default: null },
  myName:    { type: String, default: '' },
  myAvatar:  { type: String, default: '' },
})

const emit = defineEmits(['stop', 'join', 'start', 'again', 'answer-correct', 'retry'])

const { animePool, screenshotCache, loadError, isLoadingPool } = useAnimeQuiz()

// ── Derived state from quizState (server-driven) ──────────────────────────────
const phase         = computed(() => props.quizState?.phase ?? 'loading')
const players       = computed(() => props.quizState?.players ?? [])
const scores        = computed(() => props.quizState?.scores ?? {})
const currentIdx    = computed(() => props.quizState?.currentIdx ?? 0)
const totalQ        = computed(() => props.quizState?.totalQ ?? 0)
const stage         = computed(() => props.quizState?.stage ?? 0)
const stageTimeLeft = computed(() => props.quizState?.stageTimeLeft ?? 0)
const roundAnswered = computed(() => props.quizState?.roundAnswered ?? {})
const roundWinner   = computed(() => props.quizState?.roundWinner ?? '')
const revealAnime   = computed(() => props.quizState?.revealAnime ?? null)
const roundScored   = computed(() => props.quizState?.roundScored ?? [])
const animeId       = computed(() => props.quizState?.animeId ?? null)
const animePoster   = computed(() => props.quizState?.animePoster ?? null)
const maskedTitle   = computed(() => props.quizState?.maskedTitle ?? '')

const sortedScores = computed(() =>
  players.value
    .map(p => ({ ...p, points: scores.value[p.id] ?? 0 }))
    .sort((a, b) => b.points - a.points)
)

const totalTimeLeft = computed(() => {
  let t = stageTimeLeft.value
  for (let i = stage.value + 1; i < STAGE_DURATIONS.length; i++) t += STAGE_DURATIONS[i]
  return t
})

// ── Local view state ──────────────────────────────────────────────────────────
const imgError       = ref(false)
const answerDraft    = ref('')
const answerInputEl  = ref(null)
const shaking        = ref(false)
const selectedSugIdx = ref(-1)
const submitting     = ref(false)

const viewMode      = ref('screenshot')
const screenshotIdx = ref(0)

// Load screenshots whenever animeId changes
watch(animeId, id => {
  if (id) loadScreenshots(id)
}, { immediate: true })

const hasPosterSlot   = computed(() => stage.value >= 3 || !!roundWinner.value)
const availableSlides = computed(() => stage.value + 1)

const canGoBack    = computed(() => viewMode.value === 'poster' ? stage.value > 0 : screenshotIdx.value > 0)
const canGoForward = computed(() => {
  if (viewMode.value === 'poster') return false
  return screenshotIdx.value < stage.value || hasPosterSlot.value
})

function goBack() {
  if (viewMode.value === 'poster') {
    viewMode.value = 'screenshot'
    screenshotIdx.value = stage.value
  } else {
    screenshotIdx.value = Math.max(0, screenshotIdx.value - 1)
  }
}
function goForward() {
  if (viewMode.value === 'screenshot') {
    if (screenshotIdx.value < stage.value) {
      screenshotIdx.value++
    } else if (hasPosterSlot.value) {
      viewMode.value = 'poster'
    }
  }
}

watch(stage, s => {
  viewMode.value = 'screenshot'
  screenshotIdx.value = s
  imgError.value = false
})

watch(currentIdx, () => {
  viewMode.value       = 'screenshot'
  screenshotIdx.value  = 0
  imgError.value       = false
  answerDraft.value    = ''
  selectedSugIdx.value = -1
  nextTick(() => answerInputEl.value?.focus())
})

watch(phase, p => {
  if (p === 'playing') nextTick(() => answerInputEl.value?.focus())
})

// ── Image URL ─────────────────────────────────────────────────────────────────
const mainImageUrl = computed(() => {
  if (!animeId.value) return null
  if (viewMode.value === 'poster') return animePoster.value
  return getStageImage(animeId.value, animePoster.value, screenshotIdx.value)
})

const imgKey = computed(() => {
  if (!animeId.value) return 'none'
  return viewMode.value === 'poster'
    ? `${animeId.value}-poster`
    : `${animeId.value}-shot-${screenshotIdx.value}`
})

// ── Autocomplete suggestions ──────────────────────────────────────────────────
const suggestions = computed(() => {
  const q = answerDraft.value.toLowerCase().trim()
  if (q.length < 2 || isAnswered.value || !!revealAnime.value || !!roundWinner.value) return []
  const exact = [], prefix = [], partial = []
  for (const a of animePool.value) {
    const rn = (a.russian ?? '').toLowerCase()
    const en = (a.name ?? '').toLowerCase()
    if (rn === q || en === q)             { exact.push(a); continue }
    if (rn.startsWith(q) || en.startsWith(q)) { prefix.push(a); continue }
    if (rn.includes(q) || en.includes(q))     { partial.push(a) }
  }
  return [...exact, ...prefix, ...partial].slice(0, 7)
})

watch(answerDraft, () => { selectedSugIdx.value = -1 })

function selectSuggestion(anime) {
  answerDraft.value = anime.russian || anime.name
  submitDraft()
}

// ── Answer logic ──────────────────────────────────────────────────────────────
const isAnswered = computed(() => props.myId ? !!roundAnswered.value[props.myId] : false)

async function submitDraft() {
  const text = answerDraft.value.trim()
  if (!text || !props.myId || submitting.value) return
  submitting.value = true
  try {
    const result = await submitAnswer(props.myId, props.myName, text)
    if (result.correct) {
      answerDraft.value    = ''
      selectedSugIdx.value = -1
      emit('answer-correct', { name: props.myName })
    } else {
      triggerShake()
    }
  } catch {
    triggerShake()
  } finally {
    submitting.value = false
  }
}

function triggerShake() {
  shaking.value = false
  nextTick(() => {
    shaking.value = true
    setTimeout(() => { shaking.value = false }, 500)
  })
}

function onKeyDown(e) {
  const sug = suggestions.value
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    selectedSugIdx.value = Math.min(selectedSugIdx.value + 1, sug.length - 1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    selectedSugIdx.value = Math.max(selectedSugIdx.value - 1, -1)
  } else if (e.key === 'Enter') {
    e.preventDefault()
    if (selectedSugIdx.value >= 0 && sug[selectedSugIdx.value]) selectSuggestion(sug[selectedSugIdx.value])
    else submitDraft()
  } else if (e.key === 'Escape') {
    selectedSugIdx.value = -1
  }
}

// ── Labels ────────────────────────────────────────────────────────────────────
const phaseLabel = computed(() => ({
  loading: 'ЗАГРУЗКА', lobby: 'ЛОББИ', playing: 'В ЭФИРЕ', results: 'ИТОГИ', error: 'ОШИБКА',
}[phase.value] ?? ''))

const isJoined = computed(() => props.myId && players.value.some(p => p.id === props.myId))

const winnerPlayer       = computed(() => players.value.find(p => p.id === roundWinner.value) ?? null)
const winnerName         = computed(() => winnerPlayer.value?.name ?? 'Участник')
const winnerGenderSuffix = computed(() => {
  const n = winnerName.value
  return (n.endsWith('а') || n.endsWith('я')) ? 'а' : ''
})

const timerFillPct = computed(() => Math.max(0, Math.min(100, (totalTimeLeft.value / MAX_TIME) * 100)))

function rankEmoji(i) { return ['🥇', '🥈', '🥉'][i] ?? `${i + 1}.` }
</script>

<style scoped>
/* ─── Container ──────────────────────────────────────────────────────────── */
.quiz-view {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  border: 1px solid #1e1e3f;
  border-radius: 10px;
  overflow: hidden;
  background: #080816;
  box-shadow: 0 0 24px rgba(157, 78, 221, 0.08), inset 0 0 40px rgba(0, 0, 0, 0.4);
  width: 100%;
  height: 100%;
}

/* ─── Header ─────────────────────────────────────────────────────────────── */
.quiz-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 14px;
  background: rgba(157, 78, 221, 0.06);
  border-bottom: 1px solid #1e1e3f;
  flex-shrink: 0;
}
.quiz-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: 'Orbitron', sans-serif;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 2px;
  color: #c8a0f0;
  text-shadow: 0 0 10px rgba(157, 78, 221, 0.5);
  white-space: nowrap;
}
.quiz-emoji { font-size: 14px; }

.phase-badge {
  padding: 2px 7px;
  border-radius: 4px;
  font-size: 8px;
  letter-spacing: 1.5px;
  font-weight: 800;
}
.phase-badge.loading { background: rgba(157,78,221,0.12); color: #c8a0f0; border: 1px solid rgba(157,78,221,0.3); }
.phase-badge.error   { background: rgba(255,41,87,0.12);  color: #ff6b9d; border: 1px solid rgba(255,41,87,0.3); }
.phase-badge.lobby   { background: rgba(0,245,255,0.12);  color: #00f5ff; border: 1px solid rgba(0,245,255,0.3); }
.phase-badge.playing {
  background: rgba(255,107,157,0.15); color: #ff6b9d; border: 1px solid rgba(255,107,157,0.4);
  animation: badge-pulse 1.8s ease-in-out infinite;
}
.phase-badge.results { background: rgba(57,255,20,0.12); color: #39ff14; border: 1px solid rgba(57,255,20,0.3); }
@keyframes badge-pulse {
  0%,100% { box-shadow: 0 0 4px rgba(255,107,157,0.2); }
  50%      { box-shadow: 0 0 10px rgba(255,107,157,0.5); }
}

/* Progress */
.quiz-progress { display: flex; align-items: center; gap: 8px; flex: 1; }
.qnum { font-family: 'Orbitron', sans-serif; font-size: 9px; color: #7070a0; white-space: nowrap; }
.rounds-left { font-size: 9px; color: #4a4a7a; white-space: nowrap; }
.timer-track { flex: 1; height: 4px; border-radius: 2px; background: #1a1a3a; overflow: hidden; }
.timer-fill  {
  height: 100%; background: linear-gradient(90deg, #9d4edd, #ff6b9d);
  border-radius: 2px; transition: width 1s linear; box-shadow: 0 0 6px rgba(157,78,221,0.5);
}
.timer-text {
  font-family: 'Orbitron', sans-serif; font-size: 9px; color: #9d4edd; width: 24px; text-align: right;
}
.stop-btn {
  margin-left: auto; width: 24px; height: 24px; border-radius: 6px; border: 1px solid #2e2e5f;
  background: transparent; color: #50507a; display: flex; align-items: center;
  justify-content: center; cursor: pointer; transition: all 0.15s; flex-shrink: 0;
}
.stop-btn:hover { border-color: #ff2957; color: #ff2957; background: rgba(255,41,87,0.1); }

/* ─── Loading / Error center ─────────────────────────────────────────────── */
.center-content {
  flex: 1; display: flex; align-items: center; justify-content: center; padding: 24px;
}
.loading-card { display: flex; flex-direction: column; align-items: center; gap: 16px; }
.spinner {
  width: 44px; height: 44px; border-radius: 50%; border: 3px solid #1e1e3f;
  border-top-color: #9d4edd; animation: spin 0.9s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
.loading-text { font-family: 'Orbitron', sans-serif; font-size: 12px; font-weight: 700; letter-spacing: 1px; color: #c8a0f0; }
.loading-sub { font-size: 11px; color: #50507a; }
.error-card { display: flex; flex-direction: column; align-items: center; gap: 14px; max-width: 360px; text-align: center; }
.error-icon { font-size: 40px; }
.error-heading { font-family: 'Orbitron', sans-serif; font-size: 13px; font-weight: 700; color: #ff6b9d; letter-spacing: 1px; }
.error-msg { font-size: 12px; color: #7070a0; line-height: 1.5; }
.retry-btn {
  padding: 10px 22px; border-radius: 8px; border: none;
  background: linear-gradient(135deg, #9d4edd, #ff6b9d); color: #fff;
  font-size: 13px; font-weight: 700; cursor: pointer; transition: all 0.2s;
}
.retry-btn:hover { transform: translateY(-2px); box-shadow: 0 4px 14px rgba(157,78,221,0.4); }
.stop-text-btn { background: none; border: none; color: #50507a; font-size: 12px; cursor: pointer; text-decoration: underline; padding: 2px; }
.stop-text-btn:hover { color: #ff2957; }

/* ─── LOBBY ──────────────────────────────────────────────────────────────── */
.lobby-content { flex: 1; display: flex; align-items: center; justify-content: center; padding: 24px; overflow-y: auto; }
.lobby-card {
  background: rgba(157,78,221,0.07); border: 1px solid rgba(157,78,221,0.2);
  border-radius: 16px; padding: 32px; max-width: 520px; width: 100%;
  display: flex; flex-direction: column; align-items: center; gap: 16px; text-align: center;
}
.lobby-icon { font-size: 48px; animation: float 3s ease-in-out infinite; }
@keyframes float { 0%,100% { transform: translateY(0); } 50% { transform: translateY(-8px); } }
.lobby-heading { font-family: 'Orbitron', sans-serif; font-size: 15px; font-weight: 700; color: #c8a0f0; letter-spacing: 1px; }
.lobby-sub { font-size: 13px; color: #7070a0; line-height: 1.6; }
.lobby-sub strong { color: #ff6b9d; }

.player-list { width: 100%; background: rgba(0,0,0,0.3); border: 1px solid #1e1e3f; border-radius: 10px; padding: 12px; display: flex; flex-direction: column; gap: 8px; }
.player-list-title { font-family: 'Orbitron', sans-serif; font-size: 9px; font-weight: 700; letter-spacing: 2px; color: #50507a; text-align: left; padding-bottom: 6px; border-bottom: 1px solid #1e1e3f; }
.player-row { display: flex; align-items: center; gap: 8px; }
.player-avatar-sm { width: 28px; height: 28px; border-radius: 50%; background: #1e1e3f; overflow: hidden; display: flex; align-items: center; justify-content: center; font-size: 12px; font-weight: 700; color: #9d4edd; flex-shrink: 0; }
.player-avatar-sm img { width: 100%; height: 100%; object-fit: cover; }
.player-name-sm { font-size: 13px; color: #c8c8e8; flex: 1; text-align: left; }
.you-badge { font-size: 10px; color: #00f5ff; background: rgba(0,245,255,0.1); border: 1px solid rgba(0,245,255,0.25); border-radius: 4px; padding: 1px 6px; }

.lobby-actions { display: flex; gap: 10px; flex-wrap: wrap; justify-content: center; }
.join-btn { padding: 10px 22px; border-radius: 8px; border: 1px solid rgba(0,245,255,0.4); background: rgba(0,245,255,0.08); color: #00f5ff; font-size: 13px; font-weight: 600; cursor: pointer; transition: all 0.15s; }
.join-btn:hover { background: rgba(0,245,255,0.15); box-shadow: 0 0 12px rgba(0,245,255,0.2); }
.joined-badge { display: flex; align-items: center; padding: 10px 18px; border-radius: 8px; background: rgba(57,255,20,0.08); border: 1px solid rgba(57,255,20,0.3); color: #39ff14; font-size: 13px; font-weight: 600; }
.start-btn { padding: 10px 22px; border-radius: 8px; border: none; background: linear-gradient(135deg, #9d4edd, #ff6b9d); color: #fff; font-size: 13px; font-weight: 700; cursor: pointer; transition: all 0.2s; box-shadow: 0 4px 16px rgba(157,78,221,0.3); }
.start-btn:hover:not(:disabled) { transform: translateY(-2px); box-shadow: 0 6px 20px rgba(157,78,221,0.45); }
.start-btn:disabled { opacity: 0.35; cursor: not-allowed; }
.lobby-hint { font-size: 11px; color: #50507a; line-height: 1.5; }
.lobby-hint strong { color: #9d4edd; }

/* ─── PLAYING ─────────────────────────────────────────────────────────────── */
.game-content { flex: 1; display: flex; flex-direction: column; min-height: 0; position: relative; }

/* Image area */
.image-wrap {
  flex: 1; min-height: 0; position: relative; display: flex;
  align-items: center; justify-content: center; background: #030308; overflow: hidden;
}
.anime-img { max-width: 100%; max-height: 100%; object-fit: contain; display: block; }

/* Image swap transition */
.img-swap-enter-active { transition: opacity 0.35s ease, transform 0.35s ease; }
.img-swap-leave-active { transition: opacity 0.2s ease; position: absolute; }
.img-swap-enter-from   { opacity: 0; transform: scale(1.03); }
.img-swap-leave-to     { opacity: 0; }

.img-fallback { position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; font-size: 48px; color: #50507a; }
.img-fallback span:last-child { font-size: 14px; }

/* Navigation arrows */
.nav-btn {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 36px; height: 52px;
  border: none;
  background: rgba(8, 8, 22, 0.65);
  backdrop-filter: blur(4px);
  color: #c8c8e8;
  font-size: 26px;
  line-height: 1;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
  display: flex; align-items: center; justify-content: center;
  z-index: 4;
}
.nav-left  { left: 0;  border-radius: 0 8px 8px 0; }
.nav-right { right: 0; border-radius: 8px 0 0 8px; }
.nav-btn:hover { background: rgba(157,78,221,0.4); color: #fff; }

/* Slide label */
.slide-label {
  position: absolute;
  bottom: 8px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 10px;
  color: rgba(200,200,232,0.55);
  background: rgba(8,8,22,0.55);
  border-radius: 6px;
  padding: 3px 10px;
  backdrop-filter: blur(4px);
  pointer-events: none;
  white-space: nowrap;
  z-index: 3;
}

/* Stage dots */
.stage-dots { position: absolute; top: 10px; right: 12px; display: flex; gap: 5px; z-index: 3; }
.stage-dot { width: 7px; height: 7px; border-radius: 50%; background: #1e1e3f; border: 1px solid #2e2e5f; transition: all 0.3s; }
.stage-dot.active { background: #9d4edd; border-color: #9d4edd; box-shadow: 0 0 6px rgba(157,78,221,0.5); }
.stage-dot.current { background: #ff6b9d; border-color: #ff6b9d; box-shadow: 0 0 8px rgba(255,107,157,0.6); animation: dot-pulse 1s ease-in-out infinite; }
@keyframes dot-pulse { 0%,100% { transform: scale(1); } 50% { transform: scale(1.3); } }

/* Reveal overlay — classic centered dark overlay */
.reveal-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  background: rgba(5, 5, 16, 0.90);
  backdrop-filter: blur(2px);
  z-index: 5;
  padding: 24px;
}
.reveal-poster-thumb {
  width: auto;
  max-height: 140px;
  max-width: 100px;
  border-radius: 8px;
  object-fit: cover;
  box-shadow: 0 6px 24px rgba(0,0,0,0.6), 0 0 0 1px rgba(157,78,221,0.25);
  flex-shrink: 0;
}
.reveal-label { font-family: 'Orbitron', sans-serif; font-size: 11px; font-weight: 700; letter-spacing: 3px; color: #ff6b9d; text-shadow: 0 0 12px rgba(255,107,157,0.5); }
.reveal-name { font-size: 22px; font-weight: 700; color: #fff; text-align: center; padding: 0 24px; text-shadow: 0 2px 8px rgba(0,0,0,0.5); line-height: 1.3; }
.reveal-name-en { font-size: 13px; color: #7070a0; text-align: center; }
.reveal-scored { display: flex; flex-direction: column; gap: 5px; margin-top: 4px; align-items: center; min-width: 160px; }
.reveal-nobody { font-size: 12px; color: #6060a0; }
.reveal-scorer { display: flex; align-items: center; gap: 7px; font-size: 13px; background: rgba(255,255,255,0.06); border-radius: 8px; padding: 4px 12px; }
.reveal-scorer.first { background: rgba(157,78,221,0.15); }
.reveal-scorer-medal { font-size: 15px; }
.reveal-scorer-name { color: #d0d0f0; font-weight: 600; flex: 1; }
.reveal-scorer-pts { color: #9d4edd; font-size: 11px; font-weight: 700; }

.reveal-fade-enter-active { transition: opacity 0.4s, transform 0.4s; }
.reveal-fade-enter-from   { opacity: 0; transform: scale(0.95); }

/* Winner overlay */
.winner-overlay {
  position: absolute; bottom: 50%; left: 50%; transform: translate(-50%, 50%);
  display: flex; flex-direction: column; align-items: center; gap: 4px;
  background: rgba(57,255,20,0.12); border: 1px solid rgba(57,255,20,0.35);
  border-radius: 12px; padding: 12px 24px; backdrop-filter: blur(6px); z-index: 5;
}
.winner-icon { font-size: 20px; color: #39ff14; }
.winner-text { font-size: 15px; font-weight: 700; color: #39ff14; }
.winner-pts  { font-size: 11px; color: #7070a0; }
.winner-pop-enter-active { animation: winner-in 0.4s cubic-bezier(0.34, 1.56, 0.64, 1); }
.winner-pop-leave-active { transition: opacity 0.3s, transform 0.3s; }
.winner-pop-leave-to     { opacity: 0; transform: translate(-50%, 50%) scale(0.9); }
@keyframes winner-in {
  from { opacity: 0; transform: translate(-50%, 50%) scale(0.7); }
  to   { opacity: 1; transform: translate(-50%, 50%) scale(1); }
}

/* Score strip */
.score-strip { position: absolute; top: 10px; left: 10px; display: flex; flex-direction: column; gap: 5px; max-height: 60%; overflow-y: auto; z-index: 3; }
.score-row { display: flex; align-items: center; gap: 6px; background: rgba(5,5,16,0.82); border: 1px solid #1e1e3f; border-radius: 8px; padding: 5px 8px; backdrop-filter: blur(4px); transition: border-color 0.3s; }
.score-row.answered { border-color: rgba(57,255,20,0.35); background: rgba(57,255,20,0.07); }
.score-avatar { width: 20px; height: 20px; border-radius: 50%; background: #1e1e3f; overflow: hidden; display: flex; align-items: center; justify-content: center; font-size: 10px; font-weight: 700; color: #9d4edd; flex-shrink: 0; }
.score-avatar img { width: 100%; height: 100%; object-fit: cover; }
.score-name { font-size: 11px; color: #c8c8e8; flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.score-pts  { font-family: 'Orbitron', sans-serif; font-size: 10px; color: #9d4edd; font-weight: 700; }
.score-check { color: #39ff14; font-size: 11px; }

/* ─── Hint bar (masked title) ────────────────────────────────────────────── */
.hint-bar {
  flex-shrink: 0;
  display: flex;
  align-items: baseline;
  gap: 10px;
  padding: 7px 14px;
  background: rgba(157,78,221,0.05);
  border-top: 1px solid #1e1e3f;
  min-height: 34px;
}
.hint-label-sm {
  font-family: 'Orbitron', sans-serif;
  font-size: 8px;
  font-weight: 700;
  letter-spacing: 2px;
  color: #50507a;
  white-space: nowrap;
  flex-shrink: 0;
}
.hint-masked {
  font-family: 'Orbitron', monospace;
  font-size: 13px;
  color: #c8a0f0;
  letter-spacing: 2px;
  word-break: break-word;
  line-height: 1.4;
}

/* ─── Answer bar ─────────────────────────────────────────────────────────── */
.answer-bar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: rgba(8,8,20,0.92);
  border-top: 1px solid #1e1e3f;
  position: relative;
}

/* Suggestions dropdown — rendered above the input bar */
.suggestions-dropdown {
  position: absolute;
  bottom: 100%;
  left: 12px;
  right: 12px;
  background: rgba(14,14,35,0.97);
  border: 1px solid #2e2e5f;
  border-radius: 10px 10px 0 0;
  overflow: hidden;
  z-index: 10;
  backdrop-filter: blur(8px);
  box-shadow: 0 -6px 20px rgba(0,0,0,0.5);
}
.suggestion-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 14px;
  cursor: pointer;
  transition: background 0.1s;
  border-bottom: 1px solid #1a1a38;
}
.suggestion-item:last-child { border-bottom: none; }
.suggestion-item:hover,
.suggestion-item.selected { background: rgba(157,78,221,0.15); }
.sug-russian { font-size: 13px; color: #e0e0f8; font-weight: 600; }
.sug-name    { font-size: 11px; color: #50507a; }

.input-wrap { flex: 1; position: relative; min-width: 0; }
.answer-input {
  width: 100%;
  background: rgba(20,20,45,0.9);
  border: 1px solid #2e2e5f;
  border-radius: 8px;
  color: #e0e0f8;
  font-size: 14px;
  padding: 9px 14px;
  outline: none;
  font-family: inherit;
  transition: border-color 0.15s, box-shadow 0.15s;
  box-sizing: border-box;
}
.answer-input::placeholder { color: #50507a; }
.answer-input:focus { border-color: #9d4edd; box-shadow: 0 0 0 2px rgba(157,78,221,0.15); }
.answer-input.shake { border-color: #ff2957; animation: input-shake 0.45s ease; }
@keyframes input-shake {
  0%,100% { transform: translateX(0); }
  20%      { transform: translateX(-7px); }
  40%      { transform: translateX(7px); }
  60%      { transform: translateX(-4px); }
  80%      { transform: translateX(4px); }
}

.answer-submit {
  flex-shrink: 0; width: 38px; height: 38px; border-radius: 8px; border: none;
  background: linear-gradient(135deg, #9d4edd, #ff6b9d); color: #fff;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; transition: all 0.15s; box-shadow: 0 2px 10px rgba(157,78,221,0.3);
}
.answer-submit:hover:not(:disabled) { transform: scale(1.07); box-shadow: 0 4px 14px rgba(157,78,221,0.45); }
.answer-submit:disabled { opacity: 0.35; cursor: not-allowed; background: #2e2e5f; box-shadow: none; }

.answered-badge {
  display: flex; align-items: center; gap: 8px; padding: 8px 16px; border-radius: 8px;
  background: rgba(57,255,20,0.1); border: 1px solid rgba(57,255,20,0.3);
  color: #39ff14; font-size: 13px; font-weight: 600; width: 100%; justify-content: center;
}

/* ─── RESULTS ─────────────────────────────────────────────────────────────── */
.results-content { flex: 1; display: flex; align-items: center; justify-content: center; padding: 24px; overflow-y: auto; }
.results-card {
  background: rgba(157,78,221,0.07); border: 1px solid rgba(157,78,221,0.2);
  border-radius: 16px; padding: 32px; max-width: 480px; width: 100%;
  display: flex; flex-direction: column; align-items: center; gap: 20px;
}
.results-title { font-family: 'Orbitron', sans-serif; font-size: 14px; font-weight: 700; color: #c8a0f0; letter-spacing: 2px; text-shadow: 0 0 12px rgba(157,78,221,0.4); }
.results-list { width: 100%; display: flex; flex-direction: column; gap: 8px; }
.result-row { display: flex; align-items: center; gap: 12px; padding: 10px 14px; border-radius: 10px; background: rgba(0,0,0,0.3); border: 1px solid #1e1e3f; transition: all 0.2s; }
.result-row.winner { background: rgba(157,78,221,0.12); border-color: rgba(157,78,221,0.35); box-shadow: 0 0 16px rgba(157,78,221,0.15); }
.result-rank { font-size: 18px; flex-shrink: 0; width: 24px; text-align: center; }
.result-avatar { width: 34px; height: 34px; border-radius: 50%; background: #1e1e3f; overflow: hidden; display: flex; align-items: center; justify-content: center; font-size: 14px; font-weight: 700; color: #9d4edd; flex-shrink: 0; }
.result-avatar img { width: 100%; height: 100%; object-fit: cover; }
.result-name { flex: 1; font-size: 15px; color: #c8c8e8; font-weight: 600; }
.result-pts  { font-family: 'Orbitron', sans-serif; font-size: 12px; color: #9d4edd; font-weight: 700; }
.no-scores   { text-align: center; color: #50507a; font-size: 14px; padding: 16px; }
.results-actions { display: flex; gap: 10px; }
.again-btn { padding: 10px 22px; border-radius: 8px; border: none; background: linear-gradient(135deg, #9d4edd, #ff6b9d); color: #fff; font-size: 13px; font-weight: 700; cursor: pointer; transition: all 0.2s; box-shadow: 0 4px 16px rgba(157,78,221,0.3); }
.again-btn:hover { transform: translateY(-2px); box-shadow: 0 6px 20px rgba(157,78,221,0.45); }
.close-btn { padding: 10px 18px; border-radius: 8px; border: 1px solid #2e2e5f; background: rgba(30,30,63,0.5); color: #7070a0; font-size: 13px; cursor: pointer; transition: all 0.15s; }
.close-btn:hover { border-color: #ff2957; color: #ff2957; }
</style>
