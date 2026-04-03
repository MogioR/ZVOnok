<template>
  <footer class="control-bar">
    <div class="bar-inner">
      <!-- Mic toggle -->
      <button
        class="ctrl-btn"
        :class="isMuted ? 'btn-danger' : 'btn-active'"
        :title="isMuted ? 'Включить микрофон' : 'Выключить микрофон'"
        @click="$emit('toggle-mute')"
      >
        <span class="btn-icon">
          <svg v-if="!isMuted" width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3zm5.91-3c-.49 0-.9.36-.98.85C16.52 14.2 14.47 16 12 16s-4.52-1.8-4.93-4.15c-.08-.49-.49-.85-.98-.85-.61 0-1.09.54-1 1.14.49 3 2.89 5.35 5.91 5.78V20c0 .55.45 1 1 1s1-.45 1-1v-2.08c3.02-.43 5.42-2.78 5.91-5.78.1-.6-.39-1.14-1-1.14z"/>
          </svg>
          <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M19 11h-1.7c0 .74-.16 1.43-.43 2.05l1.23 1.23c.56-.98.9-2.09.9-3.28zm-4.02.17c0-.06.02-.11.02-.17V5c0-1.66-1.34-3-3-3S9 3.34 9 5v.18l5.98 5.99zM4.27 3L3 4.27l6.01 6.01V11c0 1.66 1.33 3 2.99 3 .22 0 .44-.03.65-.08l1.66 1.66c-.71.33-1.5.52-2.31.52-2.76 0-5.3-2.1-5.3-5.1H5c0 3.41 2.72 6.23 6 6.72V20c0 .55.45 1 1 1s1-.45 1-1v-3.28c.91-.13 1.77-.45 2.54-.9L19.73 21 21 19.73 4.27 3z"/>
          </svg>
        </span>
        <span class="btn-label">{{ isMuted ? 'МИК ВЫКЛ' : 'МИК ВКЛ' }}</span>
      </button>

      <!-- Screen share + settings gear -->
      <div class="share-cluster">
        <button
          class="ctrl-btn"
          :class="isScreenSharing ? 'btn-cyan-active' : 'btn-default'"
          :title="isScreenSharing ? 'Остановить трансляцию' : 'Транслировать экран'"
          @click="$emit('toggle-screen-share')"
        >
          <span class="btn-icon">
            <svg v-if="!isScreenSharing" width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <path d="M20 3H4c-1.1 0-2 .9-2 2v11c0 1.1.9 2 2 2h3l-1 1v1h12v-1l-1-1h3c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 13H4V5h16v11zM9.06 14H11v-3h2v3h1.94L12 16.94 9.06 14z"/>
            </svg>
            <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <path d="M20 3H4c-1.1 0-2 .9-2 2v11c0 1.1.9 2 2 2h3l-1 1v1h12v-1l-1-1h3c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 13H4V5h16v11zM8 10h8v2H8z"/>
            </svg>
          </span>
          <span class="btn-label">{{ isScreenSharing ? 'СТОП СТРИМ' : 'СТРИМ' }}</span>
        </button>

        <ScreenShareSettings :settings="screenShareSettings" />
      </div>

      <!-- Status picker -->
      <div class="status-wrap" ref="statusWrapEl">
        <button
          class="ctrl-btn btn-default"
          :class="{ 'btn-status-active': currentStatus }"
          title="Изменить статус"
          @click="showStatusPicker = !showStatusPicker"
        >
          <span class="btn-icon status-icon-text">{{ currentStatus ? currentStatus.emoji : '🟢' }}</span>
          <span class="btn-label">{{ currentStatus ? currentStatus.short : 'СТАТУС' }}</span>
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

      <!-- Chat toggle -->
      <button
        class="ctrl-btn btn-default"
        :class="{ 'btn-chat-active': chatOpen }"
        title="Чат"
        @click="$emit('toggle-chat')"
      >
        <span class="btn-icon chat-icon-wrap">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H5.17L4 17.17V4h16v12z"/>
          </svg>
          <span v-if="unread > 0" class="unread-badge">{{ unread > 9 ? '9+' : unread }}</span>
        </span>
        <span class="btn-label">ЧАТ</span>
      </button>

      <!-- Music -->
      <button
        class="ctrl-btn"
        :class="musicPlaying ? 'btn-music-active' : 'btn-default'"
        title="Музыкант Бот"
        @click="$emit('toggle-music')"
      >
        <span class="btn-icon music-icon-wrap">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
          </svg>
          <span v-if="musicPlaying" class="music-pulse" />
        </span>
        <span class="btn-label">МУЗЫКА</span>
      </button>

      <div class="divider" />

      <!-- Leave -->
      <button class="ctrl-btn btn-leave" title="Покинуть конференцию" @click="$emit('leave')">
        <span class="btn-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M10.09 15.59L11.5 17l5-5-5-5-1.41 1.41L12.67 11H3v2h9.67l-2.58 2.59zM19 3H5c-1.11 0-2 .9-2 2v4h2V5h14v14H5v-4H3v4c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z"/>
          </svg>
        </span>
        <span class="btn-label">ВЫЙТИ</span>
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
  isScreenSharing:     { type: Boolean, default: false },
  status:              { type: String,  default: null },
  chatOpen:            { type: Boolean, default: false },
  unread:              { type: Number,  default: 0 },
  screenShareSettings: { type: Object,  default: () => ({}) },
  musicPlaying:        { type: Boolean, default: false },
})

const emit = defineEmits(['toggle-mute', 'toggle-screen-share', 'toggle-chat', 'toggle-music', 'set-status', 'leave'])

const showStatusPicker = ref(false)
const statusWrapEl = ref(null)

const currentStatus = computed(() =>
  STATUS_OPTIONS.find((o) => o.value === props.status && o.value !== null) ?? null
)

function selectStatus(value) {
  emit('set-status', value)
  showStatusPicker.value = false
}

function onClickOutside(e) {
  if (statusWrapEl.value && !statusWrapEl.value.contains(e.target)) {
    showStatusPicker.value = false
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
  background: rgba(8, 8, 18, 0.95);
  border-top: 1px solid #1e1e3f;
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
}

.bar-inner {
  display: flex;
  align-items: center;
  gap: 8px;
}

.share-cluster {
  display: flex;
  align-items: flex-end;
  gap: 4px;
}

.ctrl-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 14px;
  border-radius: 8px;
  border: 1px solid;
  cursor: pointer;
  transition: all 0.2s;
  min-width: 76px;
  position: relative;
}
.ctrl-btn:hover { transform: translateY(-2px); }

.btn-icon {
  display: flex; align-items: center; justify-content: center;
  position: relative;
}

.btn-label {
  font-family: 'Orbitron', sans-serif;
  font-size: 7px; font-weight: 700; letter-spacing: 1.5px; white-space: nowrap;
}

/* ─── Button variants ────────────────────────────────────────────────────── */
.btn-active { background: rgba(57,255,20,0.1); border-color: #39ff14; color: #39ff14; }
.btn-active:hover { background: rgba(57,255,20,0.18); box-shadow: 0 0 16px rgba(57,255,20,0.3); }

.btn-danger { background: rgba(255,41,87,0.12); border-color: #ff2957; color: #ff2957; }
.btn-danger:hover { background: rgba(255,41,87,0.2); box-shadow: 0 0 16px rgba(255,41,87,0.3); }

.btn-default { background: rgba(30,30,63,0.5); border-color: #2e2e5f; color: #7070a0; }
.btn-default:hover { border-color: #00f5ff; color: #00f5ff; background: rgba(0,245,255,0.07); }

.btn-cyan-active {
  background: rgba(0,245,255,0.1); border-color: #00f5ff; color: #00f5ff;
  animation: share-pulse 2s ease-in-out infinite;
}
@keyframes share-pulse {
  0%, 100% { box-shadow: 0 0 6px rgba(0,245,255,0.2); }
  50%       { box-shadow: 0 0 14px rgba(0,245,255,0.4); }
}

.btn-status-active { border-color: #9d4edd; color: #c8a0f0; background: rgba(157,78,221,0.12); }
.btn-chat-active   { border-color: #9d4edd; color: #9d4edd; background: rgba(157,78,221,0.1); }

.btn-music-active {
  background: rgba(57,255,20,0.07);
  border-color: #39ff14;
  color: #39ff14;
  animation: music-glow 1.5s ease-in-out infinite;
}
@keyframes music-glow {
  0%, 100% { box-shadow: 0 0 6px rgba(57,255,20,0.2); }
  50%       { box-shadow: 0 0 16px rgba(57,255,20,0.45); }
}

.music-icon-wrap { position: relative; }
.music-pulse {
  position: absolute;
  top: -3px; right: -3px;
  width: 7px; height: 7px;
  background: #39ff14;
  border-radius: 50%;
  box-shadow: 0 0 6px #39ff14;
  animation: dot-pulse 1.2s ease-in-out infinite;
}
@keyframes dot-pulse { 0%,100% { opacity: 1; transform: scale(1); } 50% { opacity: .6; transform: scale(1.3); } }

.btn-leave { background: rgba(255,41,87,0.08); border-color: #ff2957; color: #ff2957; }
.btn-leave:hover { background: rgba(255,41,87,0.2); box-shadow: 0 0 16px rgba(255,41,87,0.3); }

/* ─── Status icon (emoji) ────────────────────────────────────────────────── */
.status-icon-text { font-size: 18px; line-height: 1; }

/* ─── Status popup ───────────────────────────────────────────────────────── */
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
  min-width: 150px;
  z-index: 200;
  box-shadow: 0 8px 24px rgba(0,0,0,0.6);
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
.status-opt:hover { background: rgba(157,78,221,0.15); }
.status-opt.active { background: rgba(157,78,221,0.2); color: #c8a0f0; }

.opt-emoji { font-size: 16px; }
.opt-label { font-size: 13px; }

/* ─── Chat unread badge ──────────────────────────────────────────────────── */
.chat-icon-wrap { position: relative; }

.unread-badge {
  position: absolute;
  top: -5px; right: -7px;
  min-width: 16px; height: 16px;
  background: #ff2957;
  border-radius: 8px;
  font-size: 9px;
  font-weight: 700;
  color: #fff;
  display: flex; align-items: center; justify-content: center;
  padding: 0 3px;
  box-shadow: 0 0 6px rgba(255,41,87,0.6);
  font-family: 'Orbitron', sans-serif;
}

.divider { width: 1px; height: 36px; background: #1e1e3f; margin: 0 4px; }

/* ─── Mobile ─────────────────────────────────────────────────────────────── */
@media (max-width: 640px) {
  .control-bar {
    /* Reserve space for Home indicator / gesture bar on iPhone */
    padding-bottom: env(safe-area-inset-bottom, 0px);
    height: calc(var(--bar-h) + env(safe-area-inset-bottom, 0px));
    min-height: calc(var(--bar-h) + env(safe-area-inset-bottom, 0px));
  }

  .bar-inner {
    width: 100%;
    padding: 0 6px;
    gap: 4px;
    justify-content: space-around;
  }

  .ctrl-btn {
    flex: 1;
    min-width: 0;
    max-width: 56px;
    padding: 8px 4px;
    gap: 0;
    border-radius: 10px;
  }

  /* Icons slightly bigger since there's no label */
  .ctrl-btn svg { width: 22px; height: 22px; }

  .btn-label { display: none; }
  .divider    { display: none; }

  /* Screen share gear hidden — can't share screen from mobile anyway */
  .share-cluster { gap: 0; }
  .share-cluster > :last-child { display: none; }
}
</style>
