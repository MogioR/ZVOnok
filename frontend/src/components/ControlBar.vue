<template>
  <footer class="control-bar">
    <div class="bar-inner">
      <!-- Звук: микрофон, глухой режим, PTT -->
      <div class="bar-group" aria-label="Звук">
        <span class="group-label">Звук</span>
        <div class="group-btns">
          <button
            class="ctrl-btn"
            :class="isMuted ? 'btn-danger' : 'btn-active'"
            :title="isMuted ? 'Включить микрофон' : 'Выключить микрофон'"
            @click="$emit('toggle-mute')"
          >
            <span class="btn-icon">
              <svg v-if="!isMuted" width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3zm5.91-3c-.49 0-.9.36-.98.85C16.52 14.2 14.47 16 12 16s-4.52-1.8-4.93-4.15c-.08-.49-.49-.85-.98-.85-.61 0-1.09.54-1 1.14.49 3 2.89 5.35 5.91 5.78V20c0 .55.45 1 1 1s1-.45 1-1v-2.08c3.02-.43 5.42-2.78 5.91-5.78.1-.6-.39-1.14-1-1.14z"/>
              </svg>
              <svg v-else width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M19 11h-1.7c0 .74-.16 1.43-.43 2.05l1.23 1.23c.56-.98.9-2.09.9-3.28zm-4.02.17c0-.06.02-.11.02-.17V5c0-1.66-1.34-3-3-3S9 3.34 9 5v.18l5.98 5.99zM4.27 3L3 4.27l6.01 6.01V11c0 1.66 1.33 3 2.99 3 .22 0 .44-.03.65-.08l1.66 1.66c-.71.33-1.5.52-2.31.52-2.76 0-5.3-2.1-5.3-5.1H5c0 3.41 2.72 6.23 6 6.72V20c0 .55.45 1 1 1s1-.45 1-1v-3.28c.91-.13 1.77-.45 2.54-.9L19.73 21 21 19.73 4.27 3z"/>
              </svg>
            </span>
            <span class="btn-label">Микрофон</span>
          </button>

          <button
            class="ctrl-btn"
            :class="isDeafened ? 'btn-deafen-active' : 'btn-default'"
            :title="isDeafened ? 'Снять тишину: звук других и микрофон' : 'Тишина: без звука других и без микрофона'"
            @click="$emit('toggle-deafen')"
          >
            <span class="btn-icon">
              <!-- Динамик выкл. — отдельно от значка микрофона -->
              <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M16.5 12c0-1.77-1.02-3.29-2.5-4.03v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51A8.463 8.463 0 0 0 21.5 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06c1.38-.31 2.63-.95 3.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4L9.91 6.09 12 8.18V4z"/>
              </svg>
            </span>
            <span class="btn-label">Тишина</span>
          </button>

          <button
            class="ctrl-btn"
            :class="pushToTalkEnabled ? (pushToTalkActive ? 'btn-ptt-live' : 'btn-ptt-active') : 'btn-default'"
            :title="pushToTalkEnabled ? 'Push to Talk: удерживайте пробел' : 'Включить Push to Talk (пробел)'"
            @click="$emit('toggle-push-to-talk')"
          >
            <span class="btn-icon">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 15c1.93 0 3.5-1.57 3.5-3.5v-4a3.5 3.5 0 1 0-7 0v4c0 1.93 1.57 3.5 3.5 3.5zm6-3.5a1 1 0 1 0-2 0 4 4 0 1 1-8 0 1 1 0 1 0-2 0 6 6 0 0 0 5 5.91V20H9a1 1 0 1 0 0 2h6a1 1 0 1 0 0-2h-2v-2.59A6 6 0 0 0 18 11.5z"/>
              </svg>
            </span>
            <span class="btn-label">PTT</span>
          </button>
        </div>
      </div>

      <div class="bar-divider" aria-hidden="true" />

      <!-- Экран -->
      <div class="bar-group" aria-label="Экран">
        <span class="group-label">Экран</span>
        <div class="group-btns share-row">
          <button
            class="ctrl-btn"
            :class="isScreenSharing ? 'btn-cyan-active' : 'btn-default'"
            :title="isScreenSharing ? 'Остановить трансляцию' : 'Транслировать экран'"
            @click="$emit('toggle-screen-share')"
          >
            <span class="btn-icon">
              <svg v-if="!isScreenSharing" width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M20 3H4c-1.1 0-2 .9-2 2v11c0 1.1.9 2 2 2h3l-1 1v1h12v-1l-1-1h3c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 13H4V5h16v11zM9.06 14H11v-3h2v3h1.94L12 16.94 9.06 14z"/>
              </svg>
              <svg v-else width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M20 3H4c-1.1 0-2 .9-2 2v11c0 1.1.9 2 2 2h3l-1 1v1h12v-1l-1-1h3c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 13H4V5h16v11zM8 10h8v2H8z"/>
              </svg>
            </span>
            <span class="btn-label">Стрим</span>
          </button>
          <ScreenShareSettings :settings="screenShareSettings" />

          <!-- Webcam -->
          <button
            class="ctrl-btn"
            :class="isWebcamActive ? 'btn-webcam-active' : 'btn-default'"
            :title="isWebcamActive ? 'Выключить веб-камеру' : 'Включить веб-камеру'"
            @click="$emit('toggle-webcam')"
          >
            <span class="btn-icon">
              <svg v-if="!isWebcamActive" width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z"/>
              </svg>
              <svg v-else width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M18 10.48V6c0-1.1-.9-2-2-2H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2v-4.48l4 3.98v-11l-4 3.98zM10 7c1.93 0 3.5 1.57 3.5 3.5S11.93 14 10 14s-3.5-1.57-3.5-3.5S8.07 7 10 7zm0 2c-.83 0-1.5.67-1.5 1.5S9.17 12 10 12s1.5-.67 1.5-1.5S10.83 9 10 9z"/>
              </svg>
            </span>
            <span class="btn-label">Камера</span>
          </button>
        </div>
      </div>

      <div class="bar-divider" aria-hidden="true" />

      <!-- Комната -->
      <div class="bar-group bar-group-wide" aria-label="Комната">
        <span class="group-label">Комната</span>
        <div class="group-btns">
          <div class="status-wrap" ref="statusWrapEl">
            <button
              class="ctrl-btn btn-default"
              :class="{ 'btn-status-active': currentStatus }"
              title="Статус"
              @click="showStatusPicker = !showStatusPicker"
            >
              <span class="btn-icon status-icon-text">{{ currentStatus ? currentStatus.emoji : '🟢' }}</span>
              <span class="btn-label">Статус</span>
            </button>
            <Transition name="popup">
              <div v-if="showStatusPicker" class="status-popup">
                <button
                  v-for="opt in STATUS_OPTIONS"
                  :key="opt.value ?? 'none'"
                  class="status-opt"
                  :class="{ active: status === opt.value }"
                  @click="selectStatus(opt.value)"
                >
                  <span class="opt-emoji">{{ opt.emoji }}</span>
                  <span class="opt-label">{{ opt.label }}</span>
                </button>
              </div>
            </Transition>
          </div>

          <button
            class="ctrl-btn btn-default"
            :class="{ 'btn-chat-active': chatOpen }"
            title="Чат"
            @click="$emit('toggle-chat')"
          >
            <span class="btn-icon chat-icon-wrap">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H5.17L4 17.17V4h16v12z"/>
              </svg>
              <span v-if="unread > 0" class="unread-badge">{{ unread > 9 ? '9+' : unread }}</span>
            </span>
            <span class="btn-label">Чат</span>
          </button>

          <button
            class="ctrl-btn"
            :class="musicPlaying ? 'btn-music-active' : 'btn-default'"
            title="Музыкант бот"
            @click="$emit('toggle-music')"
          >
            <span class="btn-icon music-icon-wrap">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
              </svg>
              <span v-if="musicPlaying" class="music-pulse" />
            </span>
            <span class="btn-label">Музыка</span>
          </button>

          <button
            class="ctrl-btn"
            :class="entertainmentOpen ? 'btn-ent-active' : (entertainmentActive ? 'btn-ent-running' : 'btn-default')"
            title="Мини-игры"
            @click="$emit('toggle-entertainment')"
          >
            <span class="btn-icon ent-icon-wrap">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M21 6H3c-1.1 0-2 .9-2 2v8c0 1.1.9 2 2 2h18c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm-10 7H8v3H6v-3H3v-2h3V8h2v3h3v2zm4.5 2c-.83 0-1.5-.67-1.5-1.5S14.67 12 15.5 12s1.5.67 1.5 1.5S16.33 15 15.5 15zm3-3c-.83 0-1.5-.67-1.5-1.5S17.67 10 18.5 10s1.5.67 1.5 1.5S19.33 12 18.5 12z"/>
              </svg>
              <span v-if="entertainmentActive && !entertainmentOpen" class="ent-pulse" />
            </span>
            <span class="btn-label">Игры</span>
          </button>
        </div>
      </div>

      <div class="bar-divider bar-divider-leave" aria-hidden="true" />

      <!-- Settings -->
      <div class="settings-wrap" ref="settingsWrapEl">
        <button
          class="ctrl-btn btn-default ctrl-leave-alone"
          :class="{ 'btn-settings-active': showSettings }"
          title="Настройки"
          @click="toggleSettings"
        >
          <span class="btn-icon">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
              <path d="M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58c.18-.14.23-.41.12-.61l-1.92-3.32c-.12-.22-.37-.29-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54c-.04-.24-.24-.41-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.09.63-.09.94s.02.64.07.94l-2.03 1.58c-.18.14-.23.41-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"/>
            </svg>
          </span>
          <span class="btn-label">Настройки</span>
        </button>

        <Transition name="popup">
          <div v-if="showSettings" class="settings-popup">
            <div class="settings-title">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" style="opacity:.6">
                <path d="M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58c.18-.14.23-.41.12-.61l-1.92-3.32c-.12-.22-.37-.29-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54c-.04-.24-.24-.41-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.09.63-.09.94s.02.64.07.94l-2.03 1.58c-.18.14-.23.41-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"/>
              </svg>
              УСТРОЙСТВА
            </div>

            <!-- Microphone -->
            <div class="settings-row">
              <label class="settings-label">
                <svg width="11" height="11" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3zm5.91-3c-.49 0-.9.36-.98.85C16.52 14.2 14.47 16 12 16s-4.52-1.8-4.93-4.15c-.08-.49-.49-.85-.98-.85-.61 0-1.09.54-1 1.14.49 3 2.89 5.35 5.91 5.78V20c0 .55.45 1 1 1s1-.45 1-1v-2.08c3.02-.43 5.42-2.78 5.91-5.78.1-.6-.39-1.14-1-1.14z"/>
                </svg>
                Микрофон
              </label>
              <select
                class="settings-select"
                :value="selectedAudioInput"
                @change="onAudioInputChange"
              >
                <option value="">По умолчанию</option>
                <option v-for="d in audioInputs" :key="d.deviceId" :value="d.deviceId">
                  {{ d.label || 'Микрофон ' + d.deviceId.slice(0, 6) }}
                </option>
              </select>
            </div>

            <!-- Speaker -->
            <div class="settings-row">
              <label class="settings-label">
                <svg width="11" height="11" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02z"/>
                </svg>
                Динамики
              </label>
              <select
                v-if="audioOutputs.length > 0"
                class="settings-select"
                :value="selectedAudioOutput"
                @change="onAudioOutputChange"
              >
                <option value="">По умолчанию</option>
                <option v-for="d in audioOutputs" :key="d.deviceId" :value="d.deviceId">
                  {{ d.label || 'Динамик ' + d.deviceId.slice(0, 6) }}
                </option>
              </select>
              <span v-else class="settings-no-devices">Нет доступных</span>
            </div>
          </div>
        </Transition>
      </div>

      <button class="ctrl-btn btn-leave ctrl-leave-alone" title="Покинуть конференцию" @click="$emit('leave')">
        <span class="btn-icon">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
            <path d="M10.09 15.59L11.5 17l5-5-5-5-1.41 1.41L12.67 11H3v2h9.67l-2.58 2.59zM19 3H5c-1.11 0-2 .9-2 2v4h2V5h14v14H5v-4H3v4c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z"/>
          </svg>
        </span>
        <span class="btn-label">Выйти</span>
      </button>
    </div>
  </footer>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import ScreenShareSettings from './ScreenShareSettings.vue'

const STATUS_OPTIONS = [
  { value: null,    emoji: '🟢', label: 'В сети',       short: 'В СЕТИ' },
  { value: 'away',  emoji: '☕', label: 'Отошёл',       short: 'ОТОШЁЛ' },
  { value: 'call',  emoji: '📞', label: 'Созвон',       short: 'СОЗВОН' },
  { value: 'anime', emoji: '🎌', label: 'Смотрю аниме', short: 'АНИМЕ' },
]

const props = defineProps({
  isMuted:             { type: Boolean, default: false },
  isDeafened:          { type: Boolean, default: false },
  isScreenSharing:     { type: Boolean, default: false },
  isWebcamActive:      { type: Boolean, default: false },
  status:              { type: String,  default: null },
  chatOpen:            { type: Boolean, default: false },
  unread:              { type: Number,  default: 0 },
  screenShareSettings: { type: Object,  default: () => ({}) },
  musicPlaying:        { type: Boolean, default: false },
  entertainmentOpen:   { type: Boolean, default: false },
  entertainmentActive: { type: Boolean, default: false },
  pushToTalkEnabled:   { type: Boolean, default: false },
  pushToTalkActive:    { type: Boolean, default: false },
  audioInputDeviceId:  { type: String,  default: null },
  audioOutputDeviceId: { type: String,  default: null },
})

const emit = defineEmits([
  'toggle-mute', 'toggle-deafen', 'toggle-push-to-talk',
  'toggle-screen-share', 'toggle-webcam',
  'toggle-chat', 'toggle-music', 'toggle-entertainment', 'set-status', 'leave',
  'set-audio-input', 'set-audio-output',
])

// ─── Status picker ────────────────────────────────────────────────────────────
const showStatusPicker = ref(false)
const statusWrapEl = ref(null)

const currentStatus = computed(() =>
  STATUS_OPTIONS.find((o) => o.value === props.status && o.value !== null) ?? null
)

function selectStatus(value) {
  emit('set-status', value)
  showStatusPicker.value = false
}

// ─── Settings popup ───────────────────────────────────────────────────────────
const showSettings   = ref(false)
const settingsWrapEl = ref(null)
const audioInputs    = ref([])
const audioOutputs   = ref([])
const selectedAudioInput  = ref(props.audioInputDeviceId ?? '')
const selectedAudioOutput = ref(props.audioOutputDeviceId ?? '')

async function enumerateDevices() {
  try {
    // Request permission first so labels are populated
    await navigator.mediaDevices.getUserMedia({ audio: true }).then((s) => s.getTracks().forEach((t) => t.stop())).catch(() => {})
    const devices = await navigator.mediaDevices.enumerateDevices()
    audioInputs.value  = devices.filter((d) => d.kind === 'audioinput')
    audioOutputs.value = devices.filter((d) => d.kind === 'audiooutput')
  } catch {}
}

async function toggleSettings() {
  showSettings.value = !showSettings.value
  if (showSettings.value) await enumerateDevices()
}

function onAudioInputChange(e) {
  selectedAudioInput.value = e.target.value
  emit('set-audio-input', e.target.value || null)
}

function onAudioOutputChange(e) {
  selectedAudioOutput.value = e.target.value
  emit('set-audio-output', e.target.value || null)
}

function onClickOutside(e) {
  if (statusWrapEl.value && !statusWrapEl.value.contains(e.target)) {
    showStatusPicker.value = false
  }
  if (settingsWrapEl.value && !settingsWrapEl.value.contains(e.target)) {
    showSettings.value = false
  }
}

onMounted(() => document.addEventListener('click', onClickOutside))
onBeforeUnmount(() => document.removeEventListener('click', onClickOutside))
</script>

<style scoped>
.control-bar {
  position: relative;
  z-index: 10;
  height: var(--bar-h);
  min-height: var(--bar-h);
  background: linear-gradient(180deg, rgba(14, 14, 32, 0.97) 0%, rgba(6, 6, 16, 0.99) 100%);
  border-top: 1px solid rgba(46, 46, 95, 0.85);
  backdrop-filter: blur(12px);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 -8px 32px rgba(0, 0, 0, 0.35);
}

.bar-inner {
  display: flex;
  align-items: stretch;
  gap: 0;
  max-width: 1200px;
  width: 100%;
  padding: 0 12px;
  justify-content: center;
}

.bar-group {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 6px 10px 8px;
  border-radius: 12px;
  background: rgba(18, 18, 42, 0.45);
  border: 1px solid rgba(36, 36, 72, 0.9);
}

.bar-group-wide {
  flex: 1;
  min-width: 0;
  max-width: 520px;
}

.group-label {
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: #5a5a8a;
  font-family: 'Orbitron', sans-serif;
  line-height: 1;
}

.group-btns {
  display: flex;
  align-items: flex-end;
  gap: 6px;
  flex-wrap: wrap;
  justify-content: center;
}

.share-row {
  align-items: flex-end;
}

.bar-divider {
  width: 1px;
  align-self: stretch;
  min-height: 36px;
  margin: 10px 8px 0;
  background: linear-gradient(180deg, transparent, #2e2e5f 20%, #2e2e5f 80%, transparent);
  flex-shrink: 0;
}

.bar-divider-leave {
  margin-top: 18px;
}

.ctrl-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  padding: 7px 12px 6px;
  border-radius: 10px;
  border: 1px solid;
  cursor: pointer;
  transition: border-color 0.18s, background 0.18s, color 0.18s, transform 0.15s;
  min-width: 64px;
  position: relative;
  background: rgba(12, 12, 28, 0.6);
}
.ctrl-btn:hover { transform: translateY(-1px); }

.btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}

.btn-label {
  font-family: 'Orbitron', sans-serif;
  font-size: 8px;
  font-weight: 700;
  letter-spacing: 0.04em;
  white-space: nowrap;
  color: inherit;
  opacity: 0.92;
}

.btn-active { background: rgba(57, 255, 20, 0.08); border-color: #39ff14; color: #39ff14; }
.btn-active:hover { background: rgba(57, 255, 20, 0.14); box-shadow: 0 0 14px rgba(57, 255, 20, 0.22); }

.btn-danger { background: rgba(255, 41, 87, 0.1); border-color: #ff2957; color: #ff2957; }
.btn-danger:hover { background: rgba(255, 41, 87, 0.16); box-shadow: 0 0 14px rgba(255, 41, 87, 0.25); }

.btn-deafen-active {
  background: rgba(255, 171, 64, 0.12);
  border-color: #ffab40;
  color: #ffcc80;
  box-shadow: 0 0 12px rgba(255, 171, 64, 0.2);
}
.btn-deafen-active:hover {
  box-shadow: 0 0 16px rgba(255, 171, 64, 0.32);
}

.btn-default { border-color: #2e2e5f; color: #7a7aa8; }
.btn-default:hover { border-color: #00f5ff; color: #9ee8ff; background: rgba(0, 245, 255, 0.06); }

.btn-ptt-active {
  background: rgba(157, 78, 221, 0.1);
  border-color: #9d4edd;
  color: #c8a0f0;
}
.btn-ptt-live {
  background: rgba(0, 245, 255, 0.12);
  border-color: #00f5ff;
  color: #00f5ff;
  box-shadow: 0 0 12px rgba(0, 245, 255, 0.2);
}

.btn-cyan-active {
  background: rgba(0, 245, 255, 0.1);
  border-color: #00f5ff;
  color: #00f5ff;
  animation: share-pulse 2s ease-in-out infinite;
}
@keyframes share-pulse {
  0%, 100% { box-shadow: 0 0 5px rgba(0, 245, 255, 0.15); }
  50%       { box-shadow: 0 0 12px rgba(0, 245, 255, 0.35); }
}

.btn-status-active { border-color: #9d4edd; color: #d4b8f0; background: rgba(157, 78, 221, 0.1); }
.btn-chat-active   { border-color: #9d4edd; color: #c8a0f0; background: rgba(157, 78, 221, 0.08); }

.btn-music-active {
  background: rgba(57, 255, 20, 0.06);
  border-color: #39ff14;
  color: #7dff9a;
  animation: music-glow 1.5s ease-in-out infinite;
}
@keyframes music-glow {
  0%, 100% { box-shadow: 0 0 5px rgba(57, 255, 20, 0.15); }
  50%       { box-shadow: 0 0 14px rgba(57, 255, 20, 0.35); }
}

.music-icon-wrap { position: relative; }
.music-pulse {
  position: absolute;
  top: -3px;
  right: -3px;
  width: 7px;
  height: 7px;
  background: #39ff14;
  border-radius: 50%;
  box-shadow: 0 0 6px #39ff14;
  animation: dot-pulse 1.2s ease-in-out infinite;
}
@keyframes dot-pulse { 0%, 100% { opacity: 1; transform: scale(1); } 50% { opacity: 0.65; transform: scale(1.25); } }

.btn-leave {
  align-self: center;
  margin-top: 10px;
  background: rgba(255, 41, 87, 0.08);
  border-color: #ff2957;
  color: #ff7a9a;
}
.btn-leave:hover { background: rgba(255, 41, 87, 0.18); box-shadow: 0 0 14px rgba(255, 41, 87, 0.28); }

.ctrl-leave-alone {
  margin-top: 18px;
}

.btn-ent-active {
  background: rgba(157, 78, 221, 0.1);
  border-color: #c8a0f0;
  color: #dcc4f5;
}
.btn-ent-running {
  background: rgba(255, 107, 157, 0.08);
  border-color: #ff6b9d;
  color: #ffa8c8;
  animation: ent-glow 2s ease-in-out infinite;
}
@keyframes ent-glow {
  0%, 100% { box-shadow: 0 0 4px rgba(255, 107, 157, 0.12); }
  50%      { box-shadow: 0 0 10px rgba(255, 107, 157, 0.35); }
}

.btn-webcam-active {
  background: rgba(255, 171, 64, 0.1);
  border-color: #ffab40;
  color: #ffcc80;
  animation: webcam-glow 2s ease-in-out infinite;
}
@keyframes webcam-glow {
  0%, 100% { box-shadow: 0 0 5px rgba(255, 171, 64, 0.15); }
  50%       { box-shadow: 0 0 14px rgba(255, 171, 64, 0.38); }
}

.btn-settings-active {
  border-color: #9d4edd;
  color: #c8a0f0;
  background: rgba(157, 78, 221, 0.1);
}

/* ─── Settings popup ──────────────────────────────────────────────────────── */
.settings-wrap { position: relative; }

.settings-popup {
  position: absolute;
  bottom: calc(100% + 8px);
  right: 0;
  background: rgba(10, 10, 26, 0.98);
  border: 1px solid #2e2e5f;
  border-radius: 10px;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 280px;
  z-index: 200;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.6);
}

.settings-title {
  display: flex;
  align-items: center;
  gap: 7px;
  font-family: 'Orbitron', sans-serif;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 2px;
  color: #7070a0;
  padding-bottom: 8px;
  border-bottom: 1px solid #1a1a3a;
}

.settings-row {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.settings-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-family: 'Orbitron', sans-serif;
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 1.5px;
  color: #7070a0;
  text-transform: uppercase;
}

.settings-select {
  background: #080812;
  border: 1px solid #1e1e3f;
  border-radius: 6px;
  padding: 7px 10px;
  color: #c8c8e8;
  font-family: 'Rajdhani', sans-serif;
  font-size: 13px;
  font-weight: 500;
  outline: none;
  cursor: pointer;
  transition: border-color 0.2s;
  width: 100%;
}

.settings-select:focus { border-color: #9d4edd; }

.settings-no-devices {
  font-size: 11px;
  color: #404060;
  font-style: italic;
  padding: 4px 2px;
}

.ent-icon-wrap { position: relative; }
.ent-pulse {
  position: absolute;
  top: -3px;
  right: -3px;
  width: 7px;
  height: 7px;
  background: #ff6b9d;
  border-radius: 50%;
  box-shadow: 0 0 6px #ff6b9d;
  animation: dot-pulse 1.2s ease-in-out infinite;
}

.status-icon-text { font-size: 20px; line-height: 1; }

.status-wrap { position: relative; }

.status-popup {
  position: absolute;
  bottom: calc(100% + 8px);
  left: 50%;
  transform: translateX(-50%);
  background: rgba(10, 10, 26, 0.98);
  border: 1px solid #2e2e5f;
  border-radius: 10px;
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 160px;
  z-index: 200;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.55);
}

.popup-enter-active, .popup-leave-active { transition: opacity 0.15s, transform 0.15s; }
.popup-enter-from, .popup-leave-to { opacity: 0; transform: translateX(-50%) translateY(6px); }

.status-opt {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  border: none;
  background: none;
  cursor: pointer;
  border-radius: 7px;
  color: #c8c8e8;
  transition: background 0.1s;
  width: 100%;
  text-align: left;
}
.status-opt:hover { background: rgba(157, 78, 221, 0.15); }
.status-opt.active { background: rgba(157, 78, 221, 0.2); color: #c8a0f0; }

.opt-emoji { font-size: 16px; }
.opt-label { font-size: 13px; }

.chat-icon-wrap { position: relative; }

.unread-badge {
  position: absolute;
  top: -6px;
  right: -8px;
  min-width: 16px;
  height: 16px;
  background: #ff2957;
  border-radius: 8px;
  font-size: 9px;
  font-weight: 700;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 3px;
  box-shadow: 0 0 6px rgba(255, 41, 87, 0.55);
  font-family: 'Orbitron', sans-serif;
}

@media (max-width: 900px) {
  .group-label { display: none; }
  .bar-group { padding-top: 4px; }
}

@media (max-width: 640px) {
  .control-bar {
    padding-bottom: env(safe-area-inset-bottom, 0px);
    height: calc(var(--bar-h) + env(safe-area-inset-bottom, 0px));
    min-height: calc(var(--bar-h) + env(safe-area-inset-bottom, 0px));
  }

  .bar-inner {
    width: 100%;
    padding: 0 4px;
    gap: 0;
    flex-wrap: wrap;
    justify-content: space-between;
    row-gap: 4px;
  }

  .bar-group,
  .bar-group-wide {
    flex-direction: row;
    flex-wrap: wrap;
    flex: 1 1 auto;
    min-width: 0;
    background: transparent;
    border: none;
    padding: 2px 2px;
  }

  .bar-divider { display: none; }

  .group-btns {
    flex: 1;
    justify-content: space-around;
    gap: 3px;
  }

  .ctrl-btn {
    flex: 1;
    min-width: 0;
    max-width: 56px;
    padding: 8px 4px 5px;
    gap: 0;
    border-radius: 10px;
  }

  .ctrl-btn svg { width: 22px; height: 22px; }

  .btn-label { display: none; }

  .share-row :deep(.ss-wrap) { display: none; }

  .ctrl-leave-alone {
    flex: 0 0 auto;
    max-width: 52px;
    margin-top: 0;
  }
}
</style>
