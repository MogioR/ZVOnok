<template>
  <div class="app">
    <div class="bg-grid" />
    <div class="bg-glow" />

    <JoinModal v-if="!hasJoined" :error="error" @join="handleJoin" />

    <template v-else>
      <header class="room-header">
        <div class="logo">
          <span class="logo-accent">Z</span>VOnok
        </div>

        <div class="room-info">
          <div class="status-badge" :class="connected ? 'online' : 'offline'">
            <span class="status-dot" />
            {{ connected ? 'ОНЛАЙН' : 'ПЕРЕПОДКЛЮЧЕНИЕ…' }}
          </div>
          <div class="participant-count">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
              <path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/>
            </svg>
            {{ participantsList.length + 1 }}
          </div>
        </div>

        <div v-if="error" class="error-banner">{{ error }}</div>
      </header>

      <!-- Main content: screen share fills space, participants strip at bottom -->
      <main class="room-main" :class="{ 'chat-open': chatOpen }">
        <div class="stage-area" :class="{ 'has-tabs': isAnyGameActive && allSharers.length > 0 }" :style="stageStyle">

          <!-- ── View-mode tabs (shown when game + stream both active) ─────── -->
          <div v-if="isAnyGameActive && allSharers.length > 0" class="view-tabs">
            <button :class="{ active: stageView === 'auto' }" @click="stageView = 'auto'">🎌 Игра</button>
            <button :class="{ active: stageView === 'stream' }" @click="stageView = 'stream'">📺 Стрим</button>
            <button :class="{ active: stageView === 'split' }" @click="stageView = 'split'">⊞ Оба</button>
          </div>

          <!-- ── Split layout ──────────────────────────────────────────────── -->
          <template v-if="isAnyGameActive && allSharers.length > 0 && stageView === 'split'">
            <div class="stage-split">
              <div class="split-pane">
                <AnimeQuizView
                  v-if="isAnimeQuizActive"
                  :quiz-state="quizState"
                  :my-id="myId"
                  :my-name="myInfo?.name ?? ''"
                  :my-avatar="myInfo?.avatar ?? ''"
                  @stop="handleQuizStop"
                  @join="handleQuizJoin"
                  @start="handleQuizStart"
                  @again="handleQuizAgain"
                  @answer-correct="handleQuizAnswerCorrect"
                  @retry="handleQuizRetry"
                />
                <MusicQuizView
                  v-else-if="isMusicQuizActive"
                  :music-quiz-state="musicQuizState"
                  :my-id="myId"
                  :my-name="myInfo?.name ?? ''"
                  :my-avatar="myInfo?.avatar ?? ''"
                  :volume="musicQuizVolume"
                  @stop="handleMusicQuizStop"
                  @join="handleMusicQuizJoin"
                  @start="handleMusicQuizStart"
                  @again="handleMusicQuizAgain"
                  @answer-correct="handleMusicQuizAnswerCorrect"
                />
              </div>
              <div class="split-pane">
                <ScreenShareView
                  :sharers="allSharers"
                  :active-id="activeScreenSharer"
                  :local-id="myId"
                  @select="activeScreenSharer = $event"
                  @volume-change="handleVolumeChange"
                />
              </div>
            </div>
          </template>

          <!-- ── Stream-only layout (when user picked 'stream') ────────────── -->
          <template v-else-if="isAnyGameActive && allSharers.length > 0 && stageView === 'stream'">
            <ScreenShareView
              :sharers="allSharers"
              :active-id="activeScreenSharer"
              :local-id="myId"
              @select="activeScreenSharer = $event"
              @volume-change="handleVolumeChange"
            />
          </template>

          <!-- ── Game view (default when game active, or stageView==='auto') ─ -->
          <template v-else-if="isAnimeQuizActive">
            <AnimeQuizView
              :quiz-state="quizState"
              :my-id="myId"
              :my-name="myInfo?.name ?? ''"
              :my-avatar="myInfo?.avatar ?? ''"
              @stop="handleQuizStop"
              @join="handleQuizJoin"
              @start="handleQuizStart"
              @again="handleQuizAgain"
              @answer-correct="handleQuizAnswerCorrect"
              @retry="handleQuizRetry"
            />
          </template>
          <template v-else-if="isMusicQuizActive">
            <MusicQuizView
              :music-quiz-state="musicQuizState"
              :my-id="myId"
              :my-name="myInfo?.name ?? ''"
              :my-avatar="myInfo?.avatar ?? ''"
              :volume="musicQuizVolume"
              @stop="handleMusicQuizStop"
              @join="handleMusicQuizJoin"
              @start="handleMusicQuizStart"
              @again="handleMusicQuizAgain"
              @answer-correct="handleMusicQuizAnswerCorrect"
            />
          </template>

          <!-- ── No game: show stream or placeholder ────────────────────────── -->
          <template v-else>
            <ScreenShareView
              v-if="allSharers.length > 0"
              :sharers="allSharers"
              :active-id="activeScreenSharer"
              :local-id="myId"
              @select="activeScreenSharer = $event"
              @volume-change="handleVolumeChange"
            />
            <div v-else class="stage-placeholder">
              <div class="placeholder-icon">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="currentColor" opacity="0.15">
                  <path d="M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z"/>
                </svg>
              </div>
            </div>
          </template>

        </div>

        <!-- Drag-resize handle -->
        <div
          class="resize-handle"
          @mousedown.prevent="startResize"
          title="Потяните для изменения высоты"
        />

        <!-- Participants horizontal strip -->
        <div class="participants-strip" ref="stripEl">
          <div class="participants-inner" ref="stripInnerEl">
            <ParticipantCard
              :participant="localParticipant"
              :is-local="true"
              :speaking="localSpeaking"
              :is-muted="isMuted"
            />
            <ParticipantCard
              v-for="p in participantsList"
              :key="p.id"
              :participant="p"
              :speaking="p.speaking"
              @volume-change="handleVolumeChange"
            />
            <!-- Music bot participant (virtual) -->
            <ParticipantCard
              v-if="musicBotParticipant"
              :participant="musicBotParticipant"
              :speaking="true"
              @volume-change="handleVolumeChange"
            />
            <!-- Anime quiz bot participant (virtual) -->
            <ParticipantCard
              v-if="quizBotParticipant"
              :participant="quizBotParticipant"
              :speaking="quizState?.phase === 'playing'"
            />
            <!-- Music quiz bot participant (virtual) -->
            <ParticipantCard
              v-if="musicQuizBotParticipant"
              :participant="musicQuizBotParticipant"
              :speaking="musicQuizState?.phase === 'playing'"
              @volume-change="handleVolumeChange"
            />
          </div>

          <!-- Expand button — shown when participants overflow one row -->
          <Transition name="expand-fade">
            <button
              v-if="hasOverflow"
              class="expand-btn"
              title="Показать всех участников"
              @click="showParticipantsModal = true"
            >
              <svg width="13" height="13" viewBox="0 0 24 24" fill="currentColor">
                <path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/>
              </svg>
              Все ({{ participantsList.length + 1 }})
            </button>
          </Transition>
        </div>

        <!-- Participants modal -->
        <Teleport to="body">
          <Transition name="modal">
            <div
              v-if="showParticipantsModal"
              class="participants-modal-overlay"
              @click.self="showParticipantsModal = false"
            >
              <div class="participants-modal">
                <div class="modal-header">
                  <div class="modal-title">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/>
                    </svg>
                    УЧАСТНИКИ &mdash; {{ participantsList.length + 1 }}
                  </div>
                  <button class="modal-close" @click="showParticipantsModal = false">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
                    </svg>
                  </button>
                </div>
                <div class="modal-grid">
                  <ParticipantCard
                    :participant="localParticipant"
                    :is-local="true"
                    :speaking="localSpeaking"
                    :is-muted="isMuted"
                  />
                  <ParticipantCard
                    v-for="p in participantsList"
                    :key="p.id"
                    :participant="p"
                    :speaking="p.speaking"
                    @volume-change="handleVolumeChange"
                  />
                  <ParticipantCard
                    v-if="musicBotParticipant"
                    :participant="musicBotParticipant"
                    :speaking="true"
                    @volume-change="handleVolumeChange"
                  />
                  <ParticipantCard
                    v-if="quizBotParticipant"
                    :participant="quizBotParticipant"
                    :speaking="quizState?.phase === 'playing'"
                  />
                  <ParticipantCard
                    v-if="musicQuizBotParticipant"
                    :participant="musicQuizBotParticipant"
                    :speaking="musicQuizState?.phase === 'playing'"
                    @volume-change="handleVolumeChange"
                  />
                </div>
              </div>
            </div>
          </Transition>
        </Teleport>
      </main>

      <ControlBar
        :is-muted="isMuted"
        :is-screen-sharing="isScreenSharing"
        :status="myStatus"
        :chat-open="chatOpen"
        :unread="chatUnread"
        :screen-share-settings="screenShareSettings"
        :music-playing="musicState.playing"
        :entertainment-open="entertainmentOpen"
        :entertainment-active="isAnyGameActive"
        :push-to-talk-enabled="pushToTalkEnabled"
        :push-to-talk-active="pushToTalkActive"
        @toggle-mute="toggleMute"
        @toggle-push-to-talk="handleTogglePushToTalk"
        @toggle-screen-share="handleScreenShare"
        @set-status="handleSetStatus"
        @toggle-chat="handleToggleChat"
        @toggle-music="handleToggleMusic"
        @toggle-entertainment="handleToggleEntertainment"
        @leave="handleLeave"
      />

      <ChatPanel
        :open="chatOpen"
        :messages="chatMessages"
        @close="handleCloseChat"
        @send="handleSendChat"
      />

      <MusicPanel
        :open="musicOpen"
        :state="musicState"
        @close="musicOpen = false"
      />

      <EntertainmentPanel
        :open="entertainmentOpen"
        :quiz-phase="quizState?.phase ?? 'idle'"
        :music-quiz-phase="musicQuizState?.phase ?? 'idle'"
        :music-quiz-state="musicQuizState"
        @close="entertainmentOpen = false"
        @start-quiz="handleStartQuiz"
        @open-quiz="handleOpenQuiz"
        @start-music-quiz="handleStartMusicQuiz"
        @open-music-quiz="handleOpenMusicQuiz"
      />

      <!-- Hidden audio element: streams music from server -->
      <audio
        v-if="musicState.playing"
        :ref="onMusicAudioMounted"
        src="/api/music/stream"
        autoplay
        style="display:none"
        @error="onMusicStreamError"
        @stalled="onMusicStreamStalled"
      />
    </template>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { useConference } from './composables/useConference.js'
import {
  useAnimeQuiz, openLobby as quizOpenLobby, joinLobby as quizJoinLobby,
  startGame as quizStartGame, stopGame as quizStopGame, playAgain as quizPlayAgain,
  retryLoad as quizRetryLoad,
} from './composables/useAnimeQuiz.js'
import {
  useMusicQuiz, openLobby as musicQuizOpenLobby, joinLobby as musicQuizJoinLobby,
  startGame as musicQuizStartGame, stopGame as musicQuizStopGame, playAgain as musicQuizPlayAgain,
} from './composables/useMusicQuiz.js'
import JoinModal from './components/JoinModal.vue'
import ParticipantCard from './components/ParticipantCard.vue'
import ControlBar from './components/ControlBar.vue'
import ScreenShareView from './components/ScreenShareView.vue'
import ChatPanel from './components/ChatPanel.vue'
import MusicPanel from './components/MusicPanel.vue'
import EntertainmentPanel from './components/EntertainmentPanel.vue'
import AnimeQuizView from './components/AnimeQuizView.vue'
import MusicQuizView from './components/MusicQuizView.vue'

const {
  myId, myInfo, myStatus,
  participantsList, screenSharersList,
  localStream, screenStream,
  isMuted, isScreenSharing, localSpeaking, activeScreenSharer,
  connected, error, hasJoined,
  chatMessages, chatUnread,
  screenShareSettings, musicState,
  join, leave, toggleMute, setMuted,
  setStatus, sendChatMessage, clearChatUnread, setChatOpen,
  startScreenShare, stopScreenShare, setParticipantVolume,
  quizState, musicQuizState,
} = useConference()

// Keep composable instances for local data (animePool for autocomplete, screenshots)
useAnimeQuiz()
useMusicQuiz()

// ─── Mini-games ──────────────────────────────────────────────────────────────
const entertainmentOpen = ref(false)

// 3-way view: 'auto' = game takes priority, 'stream' = show only stream, 'split' = side-by-side
const stageView = ref('auto')

const isAnimeQuizActive  = computed(() => quizState.value?.phase && quizState.value.phase !== 'idle')
const isMusicQuizActive  = computed(() => musicQuizState.value?.phase && musicQuizState.value.phase !== 'idle')
const isAnyGameActive    = computed(() => isAnimeQuizActive.value || isMusicQuizActive.value)

// ─── Controls ─────────────────────────────────────────────────────────────────
const PUSH_TO_TALK_KEY = 'zvonok_push_to_talk_v1'
const chatOpen    = ref(false)
const musicOpen   = ref(false)
const musicVolume      = ref(parseFloat(localStorage.getItem('zvonok_music_vol')      ?? '1'))
const musicQuizVolume  = ref(parseFloat(localStorage.getItem('zvonok_quiz_music_vol') ?? '1'))
const musicAudioEl = ref(null)
const pushToTalkEnabled = ref(localStorage.getItem(PUSH_TO_TALK_KEY) === '1')
const pushToTalkActive = ref(false)

let pushToTalkWasMuted = false

watch(musicVolume, (v) => {
  localStorage.setItem('zvonok_music_vol', String(v))
  if (musicAudioEl.value) musicAudioEl.value.volume = v * 0.25
})

watch(musicQuizVolume, (v) => {
  localStorage.setItem('zvonok_quiz_music_vol', String(v))
})

watch(pushToTalkEnabled, (enabled) => {
  localStorage.setItem(PUSH_TO_TALK_KEY, enabled ? '1' : '0')
  if (!enabled) stopPushToTalk()
})

// Apply volume when the audio element mounts
function onMusicAudioMounted(el) {
  if (el) {
    musicAudioEl.value = el
    el.volume = musicVolume.value * 0.25
  }
}

// If the stream errors or stalls, reconnect with a cache-busted URL so the
// browser doesn't give up on the audio element permanently.
let _musicReconnectTimer = null
function _reconnectMusicStream() {
  if (_musicReconnectTimer) return
  _musicReconnectTimer = setTimeout(() => {
    _musicReconnectTimer = null
    const el = musicAudioEl.value
    if (!el || !musicState.value?.playing) return
    const vol = el.volume
    el.src = '/api/music/stream?t=' + Date.now()
    el.volume = vol
    el.play().catch(() => {})
  }, 1500)
}
function onMusicStreamError() {
  console.warn('[music] audio stream error, reconnecting...')
  _reconnectMusicStream()
}
function onMusicStreamStalled() {
  console.warn('[music] audio stream stalled, reconnecting...')
  _reconnectMusicStream()
}

// ─── Stage resize ───────────────────────────────────────────────────────────
const stageHeight = ref(null) // null = auto (flex: 1)
const stageStyle  = computed(() =>
  stageHeight.value !== null ? { flex: 'none', height: stageHeight.value + 'px' } : {}
)

let resizeStartY = 0
let resizeStartH = 0
function startResize(e) {
  resizeStartY = e.clientY
  const el = document.querySelector('.stage-area')
  resizeStartH = el ? el.offsetHeight : 300
  document.addEventListener('mousemove', onResizeMove)
  document.addEventListener('mouseup', onResizeUp)
}
function onResizeMove(e) {
  const delta = e.clientY - resizeStartY
  const newH  = resizeStartH + delta
  // Never allow dragging participants strip off screen: keep at least strip height visible
  const mainEl  = document.querySelector('.room-main')
  const stripEl = document.querySelector('.participants-strip')
  const handleH = 6
  const maxH = mainEl && stripEl
    ? mainEl.offsetHeight - stripEl.offsetHeight - handleH - 4
    : 9999
  stageHeight.value = Math.max(80, Math.min(newH, maxH))
}
function onResizeUp() {
  document.removeEventListener('mousemove', onResizeMove)
  document.removeEventListener('mouseup', onResizeUp)
}

// ─── Participants overflow detection ────────────────────────────────────────
const stripEl      = ref(null)
const stripInnerEl = ref(null)
const hasOverflow  = ref(false)
const showParticipantsModal = ref(false)

function checkOverflow() {
  const el = stripEl.value
  if (!el) return
  hasOverflow.value = el.scrollWidth > el.clientWidth + 2
}

// ─── Real viewport height (fixes 100vh on mobile browsers) ─────────────────
function updateRealVh() {
  document.documentElement.style.setProperty('--real-100vh', `${window.innerHeight}px`)
}

let overflowObserver = null

onMounted(() => {
  updateRealVh()
  window.addEventListener('resize', updateRealVh)
  window.addEventListener('keydown', onGlobalKeydown)
  window.addEventListener('keyup', onGlobalKeyup)
  window.addEventListener('blur', stopPushToTalk)
  document.addEventListener('visibilitychange', onVisibilityChange)

  nextTick(() => {
    if (stripEl.value) {
      overflowObserver = new ResizeObserver(checkOverflow)
      overflowObserver.observe(stripEl.value)
      checkOverflow()
    }
  })
})

watch(() => participantsList.value.length, () => nextTick(checkOverflow))

onBeforeUnmount(() => {
  stopPushToTalk()
  document.removeEventListener('mousemove', onResizeMove)
  document.removeEventListener('mouseup', onResizeUp)
  document.removeEventListener('visibilitychange', onVisibilityChange)
  window.removeEventListener('resize', updateRealVh)
  window.removeEventListener('keydown', onGlobalKeydown)
  window.removeEventListener('keyup', onGlobalKeyup)
  window.removeEventListener('blur', stopPushToTalk)
  overflowObserver?.disconnect()
})

// Include local screen share so the owner can see their own stream
const allSharers = computed(() => {
  const remote = screenSharersList.value
  if (!isScreenSharing.value || !screenStream.value) return remote
  const localEntry = {
    id: myId.value,
    name: (myInfo.value?.name ?? 'Я') + ' (вы)',
    avatar: myInfo.value?.avatar ?? '',
    screenStream: screenStream.value,
    hasScreenAudio: screenStream.value?.getAudioTracks().some((t) => t.readyState === 'live'),
    volume: 1,
  }
  return [localEntry, ...remote]
})

const localParticipant = computed(() => ({
  id: myId.value,
  name: myInfo.value?.name ?? 'Я',
  avatar: myInfo.value?.avatar ?? '',
  speaking: localSpeaking.value,
  muted: isMuted.value,
  status: myStatus.value,
  volume: 1.0,
  hasScreenShare: isScreenSharing.value,
}))

// Virtual "Музыкант Бот" participant — appears when music is playing
const musicBotParticipant = computed(() => {
  if (!musicState.value?.playing) return null
  return {
    id: '__music_bot__',
    name: 'Музыкант Бот',
    avatar: '',
    speaking: true,
    muted: false,
    status: musicState.value.current?.title ?? null,
    volume: musicVolume.value,
    isBot: true,
  }
})

// Virtual "Аниме Квиз Бот" participant — appears when quiz is active
const quizBotParticipant = computed(() => {
  const phase = quizState.value?.phase
  if (!phase || phase === 'idle') return null
  const statusMap = { lobby: 'Ожидание игроков…', playing: 'Идёт игра!', results: 'Результаты' }
  return {
    id: '__quiz_bot__',
    name: 'Аниме Квиз Бот',
    avatar: '',
    speaking: phase === 'playing',
    muted: false,
    status: statusMap[phase] ?? null,
    volume: 1.0,
    isBot: true,
  }
})

// Virtual "Музыкальный Квиз Бот" participant — appears when music quiz is active
const musicQuizBotParticipant = computed(() => {
  const phase = musicQuizState.value?.phase
  if (!phase || phase === 'idle') return null
  const statusMap = { lobby: 'Ожидание игроков…', playing: 'Идёт игра!', results: 'Результаты' }
  return {
    id: '__music_quiz_bot__',
    name: 'Музыкальный Квиз Бот',
    avatar: '',
    speaking: phase === 'playing',
    muted: false,
    status: statusMap[phase] ?? null,
    volume: musicQuizVolume.value,
    isBot: true,
  }
})

// Unified volume handler: routes bots → appropriate sink, others → WebRTC audio
function handleVolumeChange(id, value) {
  if (id === '__music_bot__') {
    musicVolume.value = value
  } else if (id === '__music_quiz_bot__') {
    musicQuizVolume.value = value
  } else {
    setParticipantVolume(id, value)
  }
}

function handleToggleMusic() {
  musicOpen.value = !musicOpen.value
}

function handleToggleEntertainment() {
  entertainmentOpen.value = !entertainmentOpen.value
}

async function handleStartQuiz(settings = {}) {
  entertainmentOpen.value = false
  try {
    await quizOpenLobby(
      { id: myId.value, name: myInfo.value?.name ?? 'Участник', avatar: myInfo.value?.avatar ?? '' },
      settings,
    )
  } catch (e) {
    console.error('[App] handleStartQuiz:', e)
  }
}

function handleOpenQuiz() {
  entertainmentOpen.value = false
}

async function handleQuizJoin() {
  await quizJoinLobby({ id: myId.value, name: myInfo.value?.name ?? 'Участник', avatar: myInfo.value?.avatar ?? '' })
}

async function handleQuizStop() {
  stageView.value = 'auto'
  await quizStopGame()
}

async function handleQuizStart() {
  await quizStartGame()
}

async function handleQuizAgain() {
  await quizPlayAgain()
}

function handleQuizAnswerCorrect({ name }) {
  sendChatMessage(`🎌 ${name} угадал аниме!`)
}

async function handleQuizRetry() {
  await quizRetryLoad({ id: myId.value, name: myInfo.value?.name ?? 'Участник', avatar: myInfo.value?.avatar ?? '' })
}

// ─── Music quiz handlers ──────────────────────────────────────────────────────
async function handleStartMusicQuiz(settings = {}) {
  entertainmentOpen.value = false
  try {
    await musicQuizOpenLobby(
      { id: myId.value, name: myInfo.value?.name ?? 'Участник', avatar: myInfo.value?.avatar ?? '' },
      settings,
    )
  } catch (e) {
    console.error('[App] handleStartMusicQuiz:', e)
  }
}

function handleOpenMusicQuiz() {
  entertainmentOpen.value = false
}

async function handleMusicQuizJoin() {
  await musicQuizJoinLobby({ id: myId.value, name: myInfo.value?.name ?? 'Участник', avatar: myInfo.value?.avatar ?? '' })
}

async function handleMusicQuizStop() { stageView.value = 'auto'; await musicQuizStopGame() }
async function handleMusicQuizStart() { await musicQuizStartGame() }
async function handleMusicQuizAgain() { await musicQuizPlayAgain() }

function handleMusicQuizAnswerCorrect({ name }) {
  sendChatMessage(`🎵 ${name} угадал аниме!`)
}

async function handleJoin({ name, avatar }) {
  await join(name, avatar)
}

async function handleScreenShare() {
  if (isScreenSharing.value) await stopScreenShare()
  else await startScreenShare()
}

function handleSetStatus(status) {
  setStatus(status)
}

function handleToggleChat() {
  chatOpen.value = !chatOpen.value
  setChatOpen(chatOpen.value)
  if (chatOpen.value) clearChatUnread()
}

function handleCloseChat() {
  chatOpen.value = false
  setChatOpen(false)
}

function handleSendChat(text) {
  sendChatMessage(text)
}

function handleLeave() {
  stopPushToTalk()
  chatOpen.value = false
  setChatOpen(false)
  leave()
}

function handleTogglePushToTalk() {
  const nextEnabled = !pushToTalkEnabled.value
  pushToTalkEnabled.value = nextEnabled

  if (nextEnabled) {
    stopPushToTalk()
    if (localStream.value && !isMuted.value) setMuted(true)
  }
}

function isTypingContext(target) {
  if (!(target instanceof HTMLElement)) return false
  if (target.closest('.chat-panel')) return true
  if (target.isContentEditable || target.closest('[contenteditable="true"]')) return true
  return ['INPUT', 'TEXTAREA', 'SELECT', 'BUTTON'].includes(target.tagName)
}

function shouldStartPushToTalk(e) {
  return (
    e.code === 'Space' &&
    !e.repeat &&
    pushToTalkEnabled.value &&
    hasJoined.value &&
    !!localStream.value &&
    !pushToTalkActive.value &&
    !isTypingContext(e.target)
  )
}

function onGlobalKeydown(e) {
  if (!shouldStartPushToTalk(e)) return
  if (!isMuted.value) return

  e.preventDefault()
  pushToTalkWasMuted = true
  pushToTalkActive.value = true
  setMuted(false)
}

function onGlobalKeyup(e) {
  if (e.code !== 'Space' || !pushToTalkActive.value) return
  e.preventDefault()
  stopPushToTalk()
}

function stopPushToTalk() {
  if (!pushToTalkActive.value) return

  pushToTalkActive.value = false
  if (pushToTalkWasMuted) setMuted(true)
  pushToTalkWasMuted = false
}

function onVisibilityChange() {
  if (document.hidden) stopPushToTalk()
}
</script>

<style>
:root {
  --bg:          #080812;
  --surface:     #0f0f1e;
  --card:        #12122a;
  --border:      #1e1e3f;
  --border-bright: #2e2e5f;
  --purple:      #9d4edd;
  --cyan:        #00f5ff;
  --green:       #39ff14;
  --red:         #ff2957;
  --text:        #c8c8e8;
  --text-dim:    #7070a0;
  --text-bright: #e8e8ff;
  --header-h:    56px;
  --bar-h:       72px;
}

*, *::before, *::after { box-sizing: border-box; }

body {
  margin: 0;
  background: var(--bg);
  color: var(--text);
  font-family: 'Inter', 'Segoe UI', sans-serif;
  -webkit-font-smoothing: antialiased;
}

.app {
  position: relative;
  width: 100%;
  /* 100vh is wrong on mobile (includes browser chrome that hides/shows).
     100dvh = dynamic viewport height, tracks the actual visible area.
     The JS in onMounted sets --real-100vh as a reliable fallback. */
  height: 100vh;
  height: 100dvh;
  height: var(--real-100vh, 100dvh);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* ─── Backgrounds ────────────────────────────────────────────────────────── */
.bg-grid {
  position: fixed; inset: 0;
  background-image:
    linear-gradient(rgba(0,245,255,0.025) 1px, transparent 1px),
    linear-gradient(90deg, rgba(0,245,255,0.025) 1px, transparent 1px);
  background-size: 48px 48px;
  pointer-events: none; z-index: 0;
}
.bg-glow {
  position: fixed; inset: 0;
  background:
    radial-gradient(ellipse 60% 40% at 20% 10%, rgba(157,78,221,0.08) 0%, transparent 60%),
    radial-gradient(ellipse 50% 30% at 80% 90%, rgba(0,245,255,0.06) 0%, transparent 60%);
  pointer-events: none; z-index: 0;
}

/* ─── Header ─────────────────────────────────────────────────────────────── */
.room-header {
  position: relative; z-index: 10;
  height: var(--header-h); min-height: var(--header-h);
  display: flex; align-items: center; gap: 16px; padding: 0 20px;
  background: rgba(8,8,18,0.9);
  border-bottom: 1px solid var(--border);
  backdrop-filter: blur(8px);
}

.logo {
  font-family: 'Orbitron', sans-serif;
  font-size: 22px; font-weight: 900; letter-spacing: 2px; text-transform: uppercase;
  color: var(--text-bright);
  text-shadow: 0 0 20px rgba(157,78,221,0.5);
}
.logo-accent {
  color: var(--purple);
  text-shadow: 0 0 10px var(--purple), 0 0 30px rgba(157,78,221,0.5);
}

.room-info { margin-left: auto; display: flex; align-items: center; gap: 12px; }

.status-badge {
  display: flex; align-items: center; gap: 6px;
  padding: 4px 10px; border-radius: 4px;
  font-family: 'Orbitron', sans-serif; font-size: 10px; font-weight: 700; letter-spacing: 1.5px;
  border: 1px solid;
}
.status-badge.online  { color: var(--green); border-color: var(--green); background: rgba(57,255,20,0.08); }
.status-badge.offline { color: var(--text-dim); border-color: var(--border); background: transparent; }

.status-dot {
  width: 6px; height: 6px; border-radius: 50%; background: currentColor;
}
.status-badge.online .status-dot {
  box-shadow: 0 0 6px currentColor;
  animation: pulse-dot 2s ease-in-out infinite;
}
@keyframes pulse-dot { 0%,100% { opacity: 1; } 50% { opacity: 0.4; } }

.participant-count {
  display: flex; align-items: center; gap: 5px;
  padding: 4px 10px;
  border: 1px solid var(--border); border-radius: 4px;
  font-family: 'Orbitron', sans-serif; font-size: 12px; font-weight: 600;
  color: var(--cyan); background: rgba(0,245,255,0.05);
}

.error-banner {
  position: absolute; bottom: -36px; left: 50%; transform: translateX(-50%);
  background: rgba(255,41,87,0.15); border: 1px solid var(--red); color: var(--red);
  padding: 6px 16px; border-radius: 4px; font-size: 13px; white-space: nowrap; z-index: 20;
}

/* ─── Main ───────────────────────────────────────────────────────────────── */
.room-main {
  position: relative; z-index: 5;
  flex: 1; overflow: hidden;
  display: flex; flex-direction: column;
  transition: margin-right 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
.room-main.chat-open { margin-right: 320px; }

/* ─── Stage (screen share / placeholder) ────────────────────────────────── */
.stage-area {
  flex: 1;
  min-height: 80px;
  overflow: hidden;
  display: flex;
  align-items: stretch;
  justify-content: center;
  padding: 6px;
  position: relative;
  flex-direction: column;
  gap: 0;
}
/* Push content below the tab bar when it's visible */
.stage-area.has-tabs { padding-top: 40px; }

/* ─── Resize handle ──────────────────────────────────────────────────────── */
.resize-handle {
  flex-shrink: 0;
  height: 6px;
  cursor: row-resize;
  background: transparent;
  position: relative;
  transition: background 0.15s;
}
.resize-handle::after {
  content: '';
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 40px;
  height: 3px;
  border-radius: 2px;
  background: #2e2e5f;
  transition: background 0.15s, width 0.15s;
}
.resize-handle:hover::after {
  background: #9d4edd;
  width: 70px;
}

.stage-placeholder {
  display: flex; align-items: center; justify-content: center;
  width: 100%; height: 100%;
}

/* Split screen layout */
.stage-split {
  display: flex;
  width: 100%;
  height: 100%;
  gap: 4px;
}
.split-pane {
  flex: 1;
  min-width: 0;
  height: 100%;
  overflow: hidden;
  position: relative;
}

/* View-mode tab bar */
.view-tabs {
  position: absolute;
  top: 8px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 20;
  display: flex;
  gap: 2px;
  background: rgba(8,8,18,0.88);
  border: 1px solid rgba(157,78,221,0.3);
  border-radius: 8px;
  padding: 3px;
  backdrop-filter: blur(8px);
}
.view-tabs button {
  padding: 4px 12px;
  border: none;
  border-radius: 5px;
  background: transparent;
  color: #7070a0;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.view-tabs button:hover { background: rgba(157,78,221,0.12); color: #c8c8e8; }
.view-tabs button.active { background: rgba(157,78,221,0.2); color: #c580ff; }

/* ─── Participants strip (bottom) ────────────────────────────────────────── */
.participants-strip {
  flex-shrink: 0;
  position: relative;
  overflow: hidden;            /* no scrollbar; expand button handles overflow */
  border-top: 1px solid var(--border);
  padding: 10px 16px 12px;
}

/* Inner wrapper: centered when it fits, left-aligned when it scrolls */
.participants-inner {
  display: flex;
  gap: 10px;
  align-items: flex-end;
  width: max-content;      /* shrink to content width */
  min-width: 100%;         /* at least full strip width */
  justify-content: center; /* center when content < strip width */
}

/* ─── Expand button ──────────────────────────────────────────────────────── */
.expand-btn {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  border: 1px solid #9d4edd;
  background: rgba(8, 8, 18, 0.92);
  backdrop-filter: blur(8px);
  color: #c8a0f0;
  font-family: 'Rajdhani', sans-serif;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.15s;
  box-shadow: 0 0 12px rgba(157, 78, 221, 0.25);
  z-index: 10;
}
.expand-btn:hover {
  background: rgba(157, 78, 221, 0.15);
  box-shadow: 0 0 20px rgba(157, 78, 221, 0.4);
  color: #e0c0ff;
}
.expand-fade-enter-active, .expand-fade-leave-active { transition: opacity 0.2s, transform 0.2s; }
.expand-fade-enter-from, .expand-fade-leave-to { opacity: 0; transform: translateY(-40%) scale(0.9); }

/* ─── Participants modal ─────────────────────────────────────────────────── */
.participants-modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 500;
  background: rgba(5, 5, 16, 0.8);
  backdrop-filter: blur(6px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.participants-modal {
  background: #0d0d22;
  border: 1px solid #2e2e5f;
  border-radius: 14px;
  width: min(900px, 100%);
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 24px 60px rgba(0,0,0,0.7), 0 0 0 1px rgba(157,78,221,0.1);
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid #1e1e3f;
  flex-shrink: 0;
}

.modal-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: 'Orbitron', sans-serif;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 2px;
  color: #9d4edd;
}

.modal-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: 1px solid #2e2e5f;
  border-radius: 6px;
  background: transparent;
  color: #7070a0;
  cursor: pointer;
  transition: all 0.15s;
}
.modal-close:hover { border-color: #ff2957; color: #ff2957; }

.modal-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  padding: 18px;
  overflow-y: auto;
  justify-content: center;
  scrollbar-width: thin;
  scrollbar-color: #2e2e5f transparent;
}
.modal-grid::-webkit-scrollbar { width: 4px; }
.modal-grid::-webkit-scrollbar-thumb { background: #2e2e5f; border-radius: 2px; }

/* Modal transition */
.modal-enter-active { transition: opacity 0.2s, transform 0.2s; }
.modal-leave-active { transition: opacity 0.15s, transform 0.15s; }
.modal-enter-from   { opacity: 0; transform: scale(0.95); }
.modal-leave-to     { opacity: 0; transform: scale(0.97); }

/* ─── Mobile responsive ──────────────────────────────────────────────────── */
@media (max-width: 640px) {
  :root {
    --header-h: 48px;
    --bar-h: 58px;
  }

  /* Header: account for top notch (Dynamic Island, etc.) */
  .room-header {
    padding-top: max(0px, env(safe-area-inset-top));
    height: calc(var(--header-h) + env(safe-area-inset-top));
    min-height: calc(var(--header-h) + env(safe-area-inset-top));
  }

  /* Chat slides over content instead of pushing it */
  .room-main.chat-open { margin-right: 0; }

  /* No drag-resize on touch */
  .resize-handle { display: none; }

  /* Header compact */
  .room-header { padding: 0 10px; gap: 8px; }
  .logo { font-size: 17px; letter-spacing: 1px; }
  .status-badge { font-size: 8px; padding: 3px 7px; letter-spacing: 0.5px; }
  .participant-count { padding: 3px 7px; font-size: 11px; }

  /* Stage stays flexible */
  .stage-area { padding: 4px; min-height: 50px; }

  /* Participants strip: touch-scroll horizontally */
  .participants-strip {
    overflow-x: auto;
    overflow-y: visible;
    -webkit-overflow-scrolling: touch;
    scrollbar-width: none;
    padding: 8px 10px 10px;
  }
  .participants-strip::-webkit-scrollbar { display: none; }
  .participants-inner {
    justify-content: flex-start;
    gap: 8px;
  }

  /* Expand button repositioned for small screens */
  .expand-btn { font-size: 11px; padding: 5px 9px; }

  /* Participants modal full-screen */
  .participants-modal-overlay { padding: 0; }
  .participants-modal {
    width: 100%;
    max-height: 100%;
    border-radius: 0;
    border-left: none;
    border-right: none;
  }
  .modal-grid { gap: 8px; padding: 12px; }
}
</style>
