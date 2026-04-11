<template>
  <div class="quiz-view">

    <!-- ─── Header ──────────────────────────────────────────────────────────── -->
    <div class="quiz-header">
      <div class="quiz-title">
        <span class="quiz-emoji">🎵</span>
        МУЗЫКАЛЬНЫЙ КВИЗ
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
    <div v-if="isLoadingPool" class="center-content">
      <div class="loading-card">
        <div class="spinner" />
        <div class="loading-text">Загружаем треки из AnimeThemes…</div>
        <div class="loading-sub">опенинги и эндинги аниме</div>
      </div>
    </div>

    <!-- ═══════════════════════════════════════════════════════════════════════ -->
    <!-- ERROR                                                                  -->
    <!-- ═══════════════════════════════════════════════════════════════════════ -->
    <div v-else-if="loadError" class="center-content">
      <div class="error-card">
        <div class="error-icon">⚠️</div>
        <div class="error-heading">Не удалось загрузить треки</div>
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
        <div class="lobby-icon">🎵</div>
        <div class="lobby-heading">Угадай аниме по опенингу!</div>
        <div class="lobby-sub">
          Каждый раунд — трек из AnimeThemes.<br/>
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

      <!-- Hidden audio element — URL and key come from server state -->
      <audio
        ref="audioEl"
        :key="audioUrl"
        :src="audioUrl"
        preload="auto"
        loop
        @canplay="playAudio"
        @error="onAudioError"
      />

      <!-- Visualizer area ────────────────────────────────────────────────── -->
      <div class="visualizer-area">

        <!-- Animated waveform -->
        <div class="waveform" :class="{ paused: audioPaused, error: audioError }">
          <div
            v-for="(h, i) in WAVE_HEIGHTS"
            :key="i"
            class="wave-bar"
            :style="`--h:${h}px; --delay:${(i * 0.083) % 1}s`"
          />
        </div>

        <div class="playing-label" v-if="!audioError && !revealTheme">
          <span v-if="audioPaused" class="play-hint" @click="playAudio">▶ нажмите, чтобы воспроизвести</span>
          <span v-else>♫ УГАДАЙТЕ АНИМЕ ПО МУЗЫКЕ</span>
        </div>

        <div v-if="audioError" class="audio-error">
          ⚠ Аудио недоступно — пропускаем трек…
        </div>

        <!-- Stage hints ──────────────────────────────────────────────────── -->
        <div v-if="stage >= 1 && !revealTheme" class="stage-hints">
          <div v-if="typeBadge" class="type-badge">{{ typeBadge }}</div>
          <div v-if="maskedTitle" class="masked-title">{{ maskedTitle }}</div>
        </div>

        <!-- Score strip ──────────────────────────────────────────────────── -->
        <div v-if="players.length > 0" class="score-strip">
          <div
            v-for="p in sortedScores" :key="p.id"
            class="score-row" :class="{ answered: roundAnswered[p.id] }"
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

        <!-- Stage dots ───────────────────────────────────────────────────── -->
        <div class="stage-dots">
          <span
            v-for="i in 4" :key="i"
            class="stage-dot"
            :class="{ active: stage >= i - 1, current: stage === i - 1 }"
          />
        </div>

        <!-- Reveal overlay ───────────────────────────────────────────────── -->
        <Transition name="reveal-fade">
          <div v-if="revealTheme" class="reveal-overlay">
            <div class="reveal-label">⏰ ВРЕМЯ ВЫШЛО</div>
            <div class="reveal-anime">{{ revealTheme.animeName }}</div>
            <div class="reveal-song">
              <span class="reveal-type">{{ revealTheme.type }}{{ revealTheme.sequence }}</span>
              «{{ revealTheme.songTitle }}»
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
          <div v-if="roundWinner && !revealTheme" class="winner-overlay">
            <div class="winner-icon">✓</div>
            <div class="winner-text">{{ winnerName }} угадал{{ winnerGenderSuffix }}!</div>
            <div class="winner-pts">+3 очка</div>
          </div>
        </Transition>
      </div>

      <!-- Answer bar ──────────────────────────────────────────────────────── -->
      <div class="answer-bar" v-if="!revealTheme">

        <!-- Suggestions -->
        <div v-if="suggestions.length > 0" class="suggestions-dropdown">
          <div
            v-for="(name, i) in suggestions" :key="name"
            class="suggestion-item"
            :class="{ selected: i === selectedSugIdx }"
            @mousedown.prevent="selectSuggestion(name)"
          >
            <span class="sug-name">{{ name }}</span>
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
            v-for="(p, i) in sortedScores" :key="p.id"
            class="result-row" :class="{ winner: i === 0 }"
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
import { submitAnswer, musicThemeNames } from '../composables/useMusicQuiz.js'

const MUSIC_STAGE_DURATIONS = [15, 12, 8, 8]
const MAX_TIME = MUSIC_STAGE_DURATIONS.reduce((a, b) => a + b, 0)

const props = defineProps({
  musicQuizState: { type: Object, default: null },
  myId:     { type: String,  default: null },
  myName:   { type: String,  default: '' },
  myAvatar: { type: String,  default: '' },
  volume:   { type: Number,  default: 1 },
})

const emit = defineEmits(['stop', 'join', 'start', 'again', 'answer-correct', 'retry'])

// Pool status comes from server-broadcasted musicQuizState
const isLoadingPool = computed(() => !props.musicQuizState?.poolReady && !props.musicQuizState?.poolError)
const loadError     = computed(() => props.musicQuizState?.poolError ?? null)

// ── Derived state from musicQuizState (server-driven) ─────────────────────────
const phase         = computed(() => props.musicQuizState?.phase ?? 'loading')
const players       = computed(() => props.musicQuizState?.players ?? [])
const scores        = computed(() => props.musicQuizState?.scores ?? {})
const currentIdx    = computed(() => props.musicQuizState?.currentIdx ?? 0)
const totalQ        = computed(() => props.musicQuizState?.totalQ ?? 0)
const stage         = computed(() => props.musicQuizState?.stage ?? 0)
const stageTimeLeft = computed(() => props.musicQuizState?.stageTimeLeft ?? 0)
const roundAnswered = computed(() => props.musicQuizState?.roundAnswered ?? {})
const roundWinner   = computed(() => props.musicQuizState?.roundWinner ?? '')
const revealTheme   = computed(() => props.musicQuizState?.revealTheme ?? null)
const roundScored   = computed(() => props.musicQuizState?.roundScored ?? [])
const audioUrl      = computed(() => props.musicQuizState?.audioUrl ?? null)
const typeBadge     = computed(() => props.musicQuizState?.typeBadge ?? '')
const maskedTitle   = computed(() => props.musicQuizState?.maskedTitle ?? '')

const sortedScores = computed(() =>
  players.value
    .map(p => ({ ...p, points: scores.value[p.id] ?? 0 }))
    .sort((a, b) => b.points - a.points)
)

const totalTimeLeft = computed(() => {
  let t = stageTimeLeft.value
  for (let i = stage.value + 1; i < MUSIC_STAGE_DURATIONS.length; i++) t += MUSIC_STAGE_DURATIONS[i]
  return t
})

// ── Audio ─────────────────────────────────────────────────────────────────────
const audioEl     = ref(null)
const audioPaused = ref(false)
const audioError  = ref(false)

function applyVolume() {
  if (audioEl.value) audioEl.value.volume = props.volume
}

function playAudio() {
  if (!audioEl.value) return
  applyVolume()
  audioEl.value.play().then(() => { audioPaused.value = false }).catch(() => { audioPaused.value = true })
}

watch(() => props.volume, applyVolume)

function onAudioError() { audioError.value = true }

watch(audioUrl, () => {
  audioError.value  = false
  audioPaused.value = false
})

watch(phase, p => {
  if (p !== 'playing' && audioEl.value) audioEl.value.pause()
  if (p === 'playing') nextTick(() => answerInputEl.value?.focus())
})

watch(currentIdx, () => {
  answerDraft.value    = ''
  selectedSugIdx.value = -1
  nextTick(() => answerInputEl.value?.focus())
})

// ── Answer logic ──────────────────────────────────────────────────────────────
const answerDraft    = ref('')
const answerInputEl  = ref(null)
const shaking        = ref(false)
const selectedSugIdx = ref(-1)
const submitting     = ref(false)

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
  } catch { triggerShake() }
  finally { submitting.value = false }
}

function triggerShake() {
  shaking.value = false
  nextTick(() => { shaking.value = true; setTimeout(() => { shaking.value = false }, 500) })
}

function onKeyDown(e) {
  const sug = suggestions.value
  if (e.key === 'ArrowDown') { e.preventDefault(); selectedSugIdx.value = Math.min(selectedSugIdx.value + 1, sug.length - 1) }
  else if (e.key === 'ArrowUp') { e.preventDefault(); selectedSugIdx.value = Math.max(selectedSugIdx.value - 1, -1) }
  else if (e.key === 'Enter') { e.preventDefault(); sug[selectedSugIdx.value] ? selectSuggestion(sug[selectedSugIdx.value]) : submitDraft() }
  else if (e.key === 'Escape') { selectedSugIdx.value = -1 }
}

watch(answerDraft, () => { selectedSugIdx.value = -1 })

const suggestions = computed(() => {
  const q = answerDraft.value.toLowerCase().trim()
  if (q.length < 2 || isAnswered.value || !!revealTheme.value || !!roundWinner.value) return []
  const exact = [], prefix = [], partial = []
  for (const name of musicThemeNames.value) {
    const n = name.toLowerCase()
    if (n === q) { exact.push(name); continue }
    if (n.startsWith(q)) { prefix.push(name); continue }
    if (n.includes(q)) partial.push(name)
  }
  return [...exact, ...prefix, ...partial].slice(0, 7)
})

function selectSuggestion(name) { answerDraft.value = name; submitDraft() }

// ── Labels ────────────────────────────────────────────────────────────────────
const phaseLabel = computed(() => ({
  loading: 'ЗАГРУЗКА', lobby: 'ЛОББИ', playing: 'В ЭФИРЕ', results: 'ИТОГИ', error: 'ОШИБКА',
}[phase.value] ?? ''))

const isJoined = computed(() => props.myId && players.value.some(p => p.id === props.myId))

const winnerPlayer       = computed(() => players.value.find(p => p.id === roundWinner.value) ?? null)
const winnerName         = computed(() => winnerPlayer.value?.name ?? 'Участник')
const winnerGenderSuffix = computed(() => { const n = winnerName.value; return (n.endsWith('а') || n.endsWith('я')) ? 'а' : '' })

const timerFillPct = computed(() => Math.max(0, Math.min(100, (totalTimeLeft.value / MAX_TIME) * 100)))

function rankEmoji(i) { return ['🥇', '🥈', '🥉'][i] ?? `${i + 1}.` }

const WAVE_HEIGHTS = [12, 28, 18, 42, 22, 50, 14, 36, 26, 48, 16, 38, 24, 46, 20, 34, 10, 44, 18, 32, 26, 52, 14, 40]
</script>

<style scoped>
/* ─── Container ──────────────────────────────────────────────────────────── */
.quiz-view {
  display: flex; flex-direction: column; flex: 1; min-height: 0;
  border: 1px solid #1e1e3f; border-radius: 10px; overflow: hidden;
  background: #080816;
  box-shadow: 0 0 24px rgba(0,245,255,0.06), inset 0 0 40px rgba(0,0,0,0.4);
  width: 100%; height: 100%;
}

/* ─── Header ─────────────────────────────────────────────────────────────── */
.quiz-header {
  display: flex; align-items: center; gap: 12px; padding: 8px 14px;
  background: rgba(0,245,255,0.04); border-bottom: 1px solid #1e1e3f; flex-shrink: 0;
}
.quiz-title {
  display: flex; align-items: center; gap: 8px;
  font-family: 'Orbitron', sans-serif; font-size: 10px; font-weight: 700;
  letter-spacing: 2px; color: #80e8ff;
  text-shadow: 0 0 10px rgba(0,245,255,0.4); white-space: nowrap;
}
.quiz-emoji { font-size: 14px; }
.phase-badge {
  padding: 2px 7px; border-radius: 4px; font-size: 8px; letter-spacing: 1.5px; font-weight: 800;
}
.phase-badge.loading { background: rgba(0,245,255,0.1); color: #80e8ff; border: 1px solid rgba(0,245,255,0.25); }
.phase-badge.error   { background: rgba(255,41,87,0.12); color: #ff6b9d; border: 1px solid rgba(255,41,87,0.3); }
.phase-badge.lobby   { background: rgba(0,245,255,0.12); color: #00f5ff; border: 1px solid rgba(0,245,255,0.3); }
.phase-badge.playing {
  background: rgba(0,245,255,0.15); color: #00f5ff; border: 1px solid rgba(0,245,255,0.4);
  animation: badge-pulse 1.8s ease-in-out infinite;
}
.phase-badge.results { background: rgba(57,255,20,0.12); color: #39ff14; border: 1px solid rgba(57,255,20,0.3); }
@keyframes badge-pulse {
  0%,100% { box-shadow: 0 0 4px rgba(0,245,255,0.2); }
  50%      { box-shadow: 0 0 10px rgba(0,245,255,0.5); }
}

/* Progress */
.quiz-progress { display: flex; align-items: center; gap: 8px; flex: 1; }
.qnum { font-family: 'Orbitron', sans-serif; font-size: 9px; color: #7070a0; white-space: nowrap; }
.rounds-left { font-size: 9px; color: #3a5060; white-space: nowrap; }
.timer-track { flex: 1; height: 4px; border-radius: 2px; background: #1a1a3a; overflow: hidden; }
.timer-fill  { height: 100%; background: linear-gradient(90deg, #00b4d8, #00f5ff); border-radius: 2px; transition: width 1s linear; box-shadow: 0 0 6px rgba(0,245,255,0.4); }
.timer-text  { font-family: 'Orbitron', sans-serif; font-size: 9px; color: #00f5ff; width: 24px; text-align: right; }
.stop-btn {
  margin-left: auto; width: 24px; height: 24px; border-radius: 6px; border: 1px solid #2e2e5f;
  background: transparent; color: #50507a; display: flex; align-items: center; justify-content: center; cursor: pointer; transition: all 0.15s; flex-shrink: 0;
}
.stop-btn:hover { border-color: #ff2957; color: #ff2957; background: rgba(255,41,87,0.1); }

/* ─── Loading / Error ─────────────────────────────────────────────────────── */
.center-content { flex: 1; display: flex; align-items: center; justify-content: center; padding: 24px; }
.loading-card { display: flex; flex-direction: column; align-items: center; gap: 16px; }
.spinner { width: 44px; height: 44px; border-radius: 50%; border: 3px solid #1e1e3f; border-top-color: #00f5ff; animation: spin 0.9s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.loading-text { font-family: 'Orbitron', sans-serif; font-size: 12px; font-weight: 700; letter-spacing: 1px; color: #80e8ff; }
.loading-sub { font-size: 11px; color: #50507a; }
.error-card { display: flex; flex-direction: column; align-items: center; gap: 14px; max-width: 360px; text-align: center; }
.error-icon { font-size: 40px; }
.error-heading { font-family: 'Orbitron', sans-serif; font-size: 13px; font-weight: 700; color: #ff6b9d; letter-spacing: 1px; }
.error-msg { font-size: 12px; color: #7070a0; line-height: 1.5; }
.retry-btn { padding: 10px 22px; border-radius: 8px; border: none; background: linear-gradient(135deg, #00b4d8, #00f5ff); color: #000; font-size: 13px; font-weight: 700; cursor: pointer; transition: all 0.2s; }
.retry-btn:hover { transform: translateY(-2px); box-shadow: 0 4px 14px rgba(0,245,255,0.3); }
.stop-text-btn { background: none; border: none; color: #50507a; font-size: 12px; cursor: pointer; text-decoration: underline; padding: 2px; }
.stop-text-btn:hover { color: #ff2957; }

/* ─── LOBBY ──────────────────────────────────────────────────────────────── */
.lobby-content { flex: 1; display: flex; align-items: center; justify-content: center; padding: 24px; overflow-y: auto; }
.lobby-card {
  background: rgba(0,245,255,0.05); border: 1px solid rgba(0,245,255,0.15); border-radius: 16px;
  padding: 32px; max-width: 520px; width: 100%;
  display: flex; flex-direction: column; align-items: center; gap: 16px; text-align: center;
}
.lobby-icon { font-size: 48px; animation: float 3s ease-in-out infinite; }
@keyframes float { 0%,100% { transform: translateY(0); } 50% { transform: translateY(-8px); } }
.lobby-heading { font-family: 'Orbitron', sans-serif; font-size: 15px; font-weight: 700; color: #80e8ff; letter-spacing: 1px; }
.lobby-sub { font-size: 13px; color: #7070a0; line-height: 1.6; }
.lobby-sub strong { color: #00f5ff; }
.player-list { width: 100%; background: rgba(0,0,0,0.3); border: 1px solid #1e1e3f; border-radius: 10px; padding: 12px; display: flex; flex-direction: column; gap: 8px; }
.player-list-title { font-family: 'Orbitron', sans-serif; font-size: 9px; font-weight: 700; letter-spacing: 2px; color: #50507a; text-align: left; padding-bottom: 6px; border-bottom: 1px solid #1e1e3f; }
.player-row { display: flex; align-items: center; gap: 8px; }
.player-avatar-sm { width: 28px; height: 28px; border-radius: 50%; background: #1e1e3f; overflow: hidden; display: flex; align-items: center; justify-content: center; font-size: 12px; font-weight: 700; color: #00b4d8; flex-shrink: 0; }
.player-avatar-sm img { width: 100%; height: 100%; object-fit: cover; }
.player-name-sm { font-size: 13px; color: #c8c8e8; flex: 1; text-align: left; }
.you-badge { font-size: 10px; color: #00f5ff; background: rgba(0,245,255,0.1); border: 1px solid rgba(0,245,255,0.25); border-radius: 4px; padding: 1px 6px; }
.lobby-actions { display: flex; gap: 10px; flex-wrap: wrap; justify-content: center; }
.join-btn { padding: 10px 22px; border-radius: 8px; border: 1px solid rgba(0,245,255,0.4); background: rgba(0,245,255,0.08); color: #00f5ff; font-size: 13px; font-weight: 600; cursor: pointer; transition: all 0.15s; }
.join-btn:hover { background: rgba(0,245,255,0.15); box-shadow: 0 0 12px rgba(0,245,255,0.2); }
.joined-badge { display: flex; align-items: center; padding: 10px 18px; border-radius: 8px; background: rgba(57,255,20,0.08); border: 1px solid rgba(57,255,20,0.3); color: #39ff14; font-size: 13px; font-weight: 600; }
.start-btn { padding: 10px 22px; border-radius: 8px; border: none; background: linear-gradient(135deg, #00b4d8, #00f5ff); color: #000; font-size: 13px; font-weight: 700; cursor: pointer; transition: all 0.2s; box-shadow: 0 4px 16px rgba(0,245,255,0.25); }
.start-btn:hover:not(:disabled) { transform: translateY(-2px); box-shadow: 0 6px 20px rgba(0,245,255,0.4); }
.start-btn:disabled { opacity: 0.35; cursor: not-allowed; }
.lobby-hint { font-size: 11px; color: #50507a; line-height: 1.5; }
.lobby-hint strong { color: #00b4d8; }

/* ─── PLAYING ─────────────────────────────────────────────────────────────── */
.game-content { flex: 1; display: flex; flex-direction: column; min-height: 0; }

/* Visualizer */
.visualizer-area {
  flex: 1; min-height: 0; position: relative;
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 20px; padding: 20px; background: #030308; overflow: hidden;
}

/* Waveform bars */
.waveform {
  display: flex; align-items: center; gap: 5px;
}
.wave-bar {
  width: 5px;
  border-radius: 3px;
  background: linear-gradient(to top, #00b4d8, #00f5ff);
  box-shadow: 0 0 6px rgba(0,245,255,0.4);
  animation: wave-bounce 0.9s ease-in-out infinite;
  animation-delay: var(--delay, 0s);
}
@keyframes wave-bounce {
  0%, 100% { height: 4px; opacity: 0.4; }
  50%       { height: var(--h, 20px); opacity: 1; }
}
.waveform.paused .wave-bar { animation-play-state: paused; opacity: 0.3; }
.waveform.error  .wave-bar { background: #ff4444; box-shadow: none; opacity: 0.2; animation: none; height: 4px; }

/* Ambient glow behind waveform */
.visualizer-area::before {
  content: '';
  position: absolute;
  top: 50%; left: 50%;
  transform: translate(-50%, -50%);
  width: 300px; height: 120px;
  background: radial-gradient(ellipse, rgba(0,245,255,0.06) 0%, transparent 70%);
  pointer-events: none;
}

.playing-label {
  font-family: 'Orbitron', sans-serif;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 2px;
  color: rgba(0,245,255,0.5);
}
.play-hint {
  cursor: pointer;
  color: #00f5ff;
  text-decoration: underline;
}
.audio-error { font-size: 12px; color: #ff6b9d; }

/* Stage hints */
.stage-hints { display: flex; flex-direction: column; align-items: center; gap: 10px; }
.type-badge {
  font-family: 'Orbitron', sans-serif;
  font-size: 13px; font-weight: 800; letter-spacing: 3px;
  color: #00f5ff;
  background: rgba(0,245,255,0.1); border: 1px solid rgba(0,245,255,0.3);
  border-radius: 6px; padding: 5px 16px;
  text-shadow: 0 0 10px rgba(0,245,255,0.5);
}
.masked-title {
  font-family: 'Orbitron', monospace;
  font-size: 15px; letter-spacing: 3px; color: #c8a0f0;
  word-break: break-word; text-align: center;
}

/* Score strip */
.score-strip { position: absolute; top: 10px; left: 10px; display: flex; flex-direction: column; gap: 5px; max-height: 60%; overflow-y: auto; z-index: 3; }
.score-row { display: flex; align-items: center; gap: 6px; background: rgba(5,5,16,0.82); border: 1px solid #1e1e3f; border-radius: 8px; padding: 5px 8px; backdrop-filter: blur(4px); transition: border-color 0.3s; }
.score-row.answered { border-color: rgba(57,255,20,0.35); background: rgba(57,255,20,0.07); }
.score-avatar { width: 20px; height: 20px; border-radius: 50%; background: #1e1e3f; overflow: hidden; display: flex; align-items: center; justify-content: center; font-size: 10px; font-weight: 700; color: #00b4d8; flex-shrink: 0; }
.score-avatar img { width: 100%; height: 100%; object-fit: cover; }
.score-name { font-size: 11px; color: #c8c8e8; flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.score-pts  { font-family: 'Orbitron', sans-serif; font-size: 10px; color: #00f5ff; font-weight: 700; }
.score-check { color: #39ff14; font-size: 11px; }

/* Stage dots */
.stage-dots { position: absolute; top: 10px; right: 12px; display: flex; gap: 5px; z-index: 3; }
.stage-dot { width: 7px; height: 7px; border-radius: 50%; background: #1e1e3f; border: 1px solid #2e2e5f; transition: all 0.3s; }
.stage-dot.active { background: #00b4d8; border-color: #00b4d8; box-shadow: 0 0 6px rgba(0,180,216,0.5); }
.stage-dot.current { background: #00f5ff; border-color: #00f5ff; box-shadow: 0 0 8px rgba(0,245,255,0.7); animation: dot-pulse 1s ease-in-out infinite; }
@keyframes dot-pulse { 0%,100% { transform: scale(1); } 50% { transform: scale(1.35); } }

/* Reveal overlay */
.reveal-overlay {
  position: absolute; inset: 0; display: flex; flex-direction: column;
  align-items: center; justify-content: center; gap: 14px;
  background: rgba(3, 3, 14, 0.92); backdrop-filter: blur(2px); z-index: 5; padding: 24px;
}
.reveal-label { font-family: 'Orbitron', sans-serif; font-size: 11px; font-weight: 700; letter-spacing: 3px; color: #ff6b9d; text-shadow: 0 0 12px rgba(255,107,157,0.5); }
.reveal-anime { font-size: 26px; font-weight: 700; color: #fff; text-align: center; text-shadow: 0 2px 8px rgba(0,0,0,0.5); line-height: 1.3; }
.reveal-song { font-size: 14px; color: #80e8ff; text-align: center; display: flex; align-items: center; gap: 8px; flex-wrap: wrap; justify-content: center; }
.reveal-type { font-family: 'Orbitron', sans-serif; font-size: 10px; font-weight: 800; letter-spacing: 2px; color: #00f5ff; background: rgba(0,245,255,0.1); border: 1px solid rgba(0,245,255,0.3); border-radius: 4px; padding: 2px 8px; }
.reveal-scored { display: flex; flex-direction: column; gap: 5px; margin-top: 4px; align-items: center; min-width: 160px; }
.reveal-nobody { font-size: 12px; color: #4a6070; }
.reveal-scorer { display: flex; align-items: center; gap: 7px; font-size: 13px; background: rgba(255,255,255,0.05); border-radius: 8px; padding: 4px 12px; }
.reveal-scorer.first { background: rgba(0,180,216,0.12); }
.reveal-scorer-medal { font-size: 15px; }
.reveal-scorer-name { color: #c0d8e8; font-weight: 600; flex: 1; }
.reveal-scorer-pts { color: #00b4d8; font-size: 11px; font-weight: 700; }

.reveal-fade-enter-active { transition: opacity 0.4s, transform 0.4s; }
.reveal-fade-enter-from   { opacity: 0; transform: scale(0.95); }

/* Winner overlay */
.winner-overlay { position: absolute; bottom: 50%; left: 50%; transform: translate(-50%, 50%); display: flex; flex-direction: column; align-items: center; gap: 4px; background: rgba(57,255,20,0.12); border: 1px solid rgba(57,255,20,0.35); border-radius: 12px; padding: 12px 24px; backdrop-filter: blur(6px); z-index: 5; }
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

/* ─── Answer bar ─────────────────────────────────────────────────────────── */
.answer-bar {
  flex-shrink: 0; display: flex; align-items: center; gap: 8px; padding: 8px 12px;
  background: rgba(8,8,20,0.92); border-top: 1px solid #1e1e3f; position: relative;
}
.suggestions-dropdown {
  position: absolute; bottom: 100%; left: 12px; right: 12px;
  background: rgba(10,10,30,0.97); border: 1px solid #2e2e5f; border-radius: 10px 10px 0 0;
  overflow: hidden; z-index: 10; backdrop-filter: blur(8px); box-shadow: 0 -6px 20px rgba(0,0,0,0.5);
}
.suggestion-item { display: flex; align-items: center; gap: 10px; padding: 8px 14px; cursor: pointer; transition: background 0.1s; border-bottom: 1px solid #1a1a38; }
.suggestion-item:last-child { border-bottom: none; }
.suggestion-item:hover, .suggestion-item.selected { background: rgba(0,245,255,0.1); }
.sug-name { font-size: 13px; color: #e0e0f8; font-weight: 600; }
.sug-meta { font-size: 11px; color: #50507a; }
.input-wrap { flex: 1; min-width: 0; }
.answer-input {
  width: 100%; background: rgba(20,20,45,0.9); border: 1px solid #2e2e5f; border-radius: 8px;
  color: #e0e0f8; font-size: 14px; padding: 9px 14px; outline: none; font-family: inherit;
  transition: border-color 0.15s, box-shadow 0.15s; box-sizing: border-box;
}
.answer-input::placeholder { color: #50507a; }
.answer-input:focus { border-color: #00b4d8; box-shadow: 0 0 0 2px rgba(0,180,216,0.15); }
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
  background: linear-gradient(135deg, #00b4d8, #00f5ff); color: #000;
  display: flex; align-items: center; justify-content: center; cursor: pointer; transition: all 0.15s;
  box-shadow: 0 2px 10px rgba(0,245,255,0.25);
}
.answer-submit:hover:not(:disabled) { transform: scale(1.07); box-shadow: 0 4px 14px rgba(0,245,255,0.4); }
.answer-submit:disabled { opacity: 0.35; cursor: not-allowed; background: #2e2e5f; box-shadow: none; }
.answered-badge { display: flex; align-items: center; gap: 8px; padding: 8px 16px; border-radius: 8px; background: rgba(57,255,20,0.1); border: 1px solid rgba(57,255,20,0.3); color: #39ff14; font-size: 13px; font-weight: 600; width: 100%; justify-content: center; }

/* ─── RESULTS ─────────────────────────────────────────────────────────────── */
.results-content { flex: 1; display: flex; align-items: center; justify-content: center; padding: 24px; overflow-y: auto; }
.results-card { background: rgba(0,245,255,0.05); border: 1px solid rgba(0,245,255,0.15); border-radius: 16px; padding: 32px; max-width: 480px; width: 100%; display: flex; flex-direction: column; align-items: center; gap: 20px; }
.results-title { font-family: 'Orbitron', sans-serif; font-size: 14px; font-weight: 700; color: #80e8ff; letter-spacing: 2px; text-shadow: 0 0 12px rgba(0,245,255,0.3); }
.results-list { width: 100%; display: flex; flex-direction: column; gap: 8px; }
.result-row { display: flex; align-items: center; gap: 12px; padding: 10px 14px; border-radius: 10px; background: rgba(0,0,0,0.3); border: 1px solid #1e1e3f; transition: all 0.2s; }
.result-row.winner { background: rgba(0,245,255,0.08); border-color: rgba(0,245,255,0.3); box-shadow: 0 0 16px rgba(0,245,255,0.1); }
.result-rank { font-size: 18px; flex-shrink: 0; width: 24px; text-align: center; }
.result-avatar { width: 34px; height: 34px; border-radius: 50%; background: #1e1e3f; overflow: hidden; display: flex; align-items: center; justify-content: center; font-size: 14px; font-weight: 700; color: #00b4d8; flex-shrink: 0; }
.result-avatar img { width: 100%; height: 100%; object-fit: cover; }
.result-name { flex: 1; font-size: 15px; color: #c8c8e8; font-weight: 600; }
.result-pts  { font-family: 'Orbitron', sans-serif; font-size: 12px; color: #00f5ff; font-weight: 700; }
.no-scores   { text-align: center; color: #50507a; font-size: 14px; padding: 16px; }
.results-actions { display: flex; gap: 10px; }
.again-btn { padding: 10px 22px; border-radius: 8px; border: none; background: linear-gradient(135deg, #00b4d8, #00f5ff); color: #000; font-size: 13px; font-weight: 700; cursor: pointer; transition: all 0.2s; box-shadow: 0 4px 16px rgba(0,245,255,0.25); }
.again-btn:hover { transform: translateY(-2px); box-shadow: 0 6px 20px rgba(0,245,255,0.4); }
.close-btn { padding: 10px 18px; border-radius: 8px; border: 1px solid #2e2e5f; background: rgba(30,30,63,0.5); color: #7070a0; font-size: 13px; cursor: pointer; transition: all 0.15s; }
.close-btn:hover { border-color: #ff2957; color: #ff2957; }
</style>
