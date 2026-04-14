<template>
  <div
    class="card"
    :class="{ speaking, local: isLocal }"
    @mouseenter="hovering = true"
    @mouseleave="hovering = false"
    @click="onCardClick"
  >
    <!-- Volume overlay: hover on desktop, tap on mobile -->
    <Transition name="vol">
      <div v-if="!isLocal && (hovering || volTapped)" class="vol-overlay" @click.stop>
        <div class="vol-label">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="currentColor">
            <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02z"/>
          </svg>
          Громкость
        </div>
        <input
          type="range" min="0" max="1" step="0.01"
          :value="participant.volume ?? 1"
          class="vol-slider"
          @input="$emit('volume-change', participant.id, +$event.target.value)"
        />
        <span class="vol-pct">{{ Math.round((participant.volume ?? 1) * 100) }}%</span>
      </div>
    </Transition>

    <div class="card-inner">
      <!-- Avatar with ring -->
      <div class="avatar-wrap">
        <div class="avatar-ring" :class="{ speaking }">
          <img
            v-if="participant.avatar && !imgError"
            :src="participant.avatar"
            class="avatar-img"
            alt=""
            @error="imgError = true"
          />
          <div v-else class="avatar-fallback">{{ initials }}</div>
        </div>

        <!-- Mic badge (local = own mute state, remote = their broadcast mute state) -->
        <div
          class="mic-badge"
          :class="micMuted ? 'muted' : 'active'"
          :title="micMuted ? 'Микрофон выключен' : 'Микрофон включён'"
        >
          <svg v-if="!micMuted" width="9" height="9" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3zm5.91-3c-.49 0-.9.36-.98.85C16.52 14.2 14.47 16 12 16s-4.52-1.8-4.93-4.15c-.08-.49-.49-.85-.98-.85-.61 0-1.09.54-1 1.14.49 3 2.89 5.35 5.91 5.78V20c0 .55.45 1 1 1s1-.45 1-1v-2.08c3.02-.43 5.42-2.78 5.91-5.78.1-.6-.39-1.14-1-1.14z"/>
          </svg>
          <svg v-else width="9" height="9" viewBox="0 0 24 24" fill="currentColor">
            <path d="M19 11h-1.7c0 .74-.16 1.43-.43 2.05l1.23 1.23c.56-.98.9-2.09.9-3.28zm-4.02.17c0-.06.02-.11.02-.17V5c0-1.66-1.34-3-3-3S9 3.34 9 5v.18l5.98 5.99zM4.27 3L3 4.27l6.01 6.01V11c0 1.66 1.33 3 2.99 3 .22 0 .44-.03.65-.08l1.66 1.66c-.71.33-1.5.52-2.31.52-2.76 0-5.3-2.1-5.3-5.1H5c0 3.41 2.72 6.23 6 6.72V20c0 .55.45 1 1 1s1-.45 1-1v-3.28c.91-.13 1.77-.45 2.54-.9L19.73 21 21 19.73 4.27 3z"/>
          </svg>
        </div>

        <!-- Долгая тишина по микрофону (VAD / сигнал «говорю») -->
        <div
          v-if="micIdleLong"
          class="afk-badge"
          title="Нет сигнала голоса по микрофону более 5 минут"
        >
          💤
        </div>

        <!-- Screen share badge -->
        <div v-if="participant.hasScreenShare" class="share-badge" title="Транслирует экран">
          <svg width="9" height="9" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20 3H4c-1.1 0-2 .9-2 2v11c0 1.1.9 2 2 2h3l-1 1v1h12v-1l-1-1h3c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 13H4V5h16v11z"/>
          </svg>
        </div>
      </div>

      <!-- Name + YOU tag -->
      <div class="name-row">
        <span class="name" :title="participant.name">{{ participant.name }}</span>
        <span v-if="isLocal" class="you-tag">ВЫ</span>
      </div>

      <!-- Status badge -->
      <div v-if="currentStatus" class="status-row">
        <span class="status-chip">{{ currentStatus.emoji }} {{ currentStatus.label }}</span>
      </div>

      <!-- Speaking waveform -->
      <div class="wave-wrap" :class="{ visible: speaking && !micMuted }">
        <span v-for="i in 5" :key="i" class="wave-bar" :style="{ animationDelay: `${(i-1)*0.08}s` }" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

const hovering  = ref(false)
const volTapped = ref(false)
let   _volTimer  = null

// On desktop: hovering shows the overlay.
// On mobile (no hover): a tap toggles it, auto-hides after 4 s.
function onCardClick() {
  if (props.isLocal) return
  // If hovering is true this is a desktop mouse-click — ignore (hover handles it)
  if (hovering.value) return
  volTapped.value = !volTapped.value
  clearTimeout(_volTimer)
  if (volTapped.value) {
    _volTimer = setTimeout(() => { volTapped.value = false }, 4000)
  }
}

const STATUS_MAP = {
  away:  { emoji: '☕', label: 'Отошёл' },
  call:  { emoji: '📞', label: 'Созвон' },
  anime: { emoji: '🎌', label: 'Аниме' },
}

const props = defineProps({
  participant: { type: Object, required: true },
  isLocal:     { type: Boolean, default: false },
  speaking:    { type: Boolean, default: false },
  isMuted:     { type: Boolean, default: false },
  /** Нет активности голоса по микрофону дольше порога (см. родитель). */
  micIdleLong: { type: Boolean, default: false },
})

defineEmits(['volume-change'])

const imgError = ref(false)

const micMuted = computed(() =>
  props.isLocal ? props.isMuted : (props.participant.muted ?? false)
)

const currentStatus = computed(() =>
  STATUS_MAP[props.participant.status ?? props.participant._localStatus] ?? null
)

const initials = computed(() =>
  (props.participant.name || '?')
    .split(' ')
    .map((w) => w[0]?.toUpperCase() ?? '')
    .slice(0, 2)
    .join('')
)
</script>

<style scoped>
/* ─── Card ───────────────────────────────────────────────────────────────── */
.card {
  position: relative;
  width: 140px;
  background: #12122a;
  border: 1px solid #1e1e3f;
  border-radius: 10px;
  padding: 14px 10px 10px;
  display: flex;
  flex-direction: column;
  align-items: center;
  overflow: hidden;  /* clips vol-overlay to card bounds */
  transition: border-color 0.2s, box-shadow 0.2s, transform 0.15s;
  cursor: default;
  flex-shrink: 0;
}

.card:hover { border-color: #2e2e5f; transform: translateY(-2px); }
.card.local { border-color: #2e2e5f; background: #14142e; }
.card.speaking {
  border-color: #9d4edd;
  box-shadow: 0 0 12px rgba(157,78,221,0.3), 0 0 30px rgba(157,78,221,0.1);
}

.card-inner {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 7px;
}

/* ─── Volume overlay ─────────────────────────────────────────────────────── */
.vol-overlay {
  position: absolute;
  inset: 0;
  z-index: 20;
  border-radius: 9px;
  background: rgba(6, 6, 18, 0.92);
  backdrop-filter: blur(6px);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 16px 14px;
}

/* fade transition */
.vol-enter-active, .vol-leave-active { transition: opacity 0.15s; }
.vol-enter-from, .vol-leave-to       { opacity: 0; }

.vol-label {
  display: flex;
  align-items: center;
  gap: 5px;
  color: #9d4edd;
  font-family: 'Orbitron', sans-serif;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 1px;
}

.vol-slider {
  width: 100%;
  -webkit-appearance: none;
  height: 4px;
  border-radius: 2px;
  background: #2e2e5f;
  outline: none;
  cursor: pointer;
}
.vol-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  width: 14px; height: 14px;
  border-radius: 50%;
  background: #9d4edd;
  box-shadow: 0 0 6px rgba(157,78,221,0.8);
  cursor: pointer;
}
.vol-slider::-moz-range-thumb {
  width: 14px; height: 14px;
  border-radius: 50%;
  background: #9d4edd;
  border: none;
  cursor: pointer;
}

.vol-pct {
  font-family: 'Orbitron', sans-serif;
  font-size: 14px;
  font-weight: 700;
  color: #c8a0f0;
}

/* ─── Avatar ─────────────────────────────────────────────────────────────── */
.avatar-wrap { position: relative; }

.avatar-ring {
  width: 60px; height: 60px;
  border-radius: 50%;
  border: 2px solid #1e1e3f;
  overflow: hidden;
  display: flex; align-items: center; justify-content: center;
  background: #0a0a1a;
  transition: border-color 0.2s, box-shadow 0.3s;
}

.avatar-ring.speaking {
  border-color: #9d4edd;
  box-shadow: 0 0 0 3px rgba(157,78,221,0.25), 0 0 16px rgba(157,78,221,0.5);
  animation: pulse-speaking 1.2s ease-in-out infinite;
}

@keyframes pulse-speaking {
  0%, 100% { box-shadow: 0 0 0 3px rgba(157,78,221,0.25), 0 0 16px rgba(157,78,221,0.5); }
  50%       { box-shadow: 0 0 0 5px rgba(157,78,221,0.15), 0 0 24px rgba(157,78,221,0.7); }
}

.avatar-img { width: 100%; height: 100%; object-fit: cover; }

.avatar-fallback {
  font-family: 'Orbitron', sans-serif;
  font-size: 17px; font-weight: 700;
  color: #9d4edd;
  text-shadow: 0 0 10px rgba(157,78,221,0.5);
}

/* ─── Badges ─────────────────────────────────────────────────────────────── */
.mic-badge, .share-badge {
  position: absolute;
  bottom: -1px;
  width: 17px; height: 17px;
  border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  border: 1.5px solid #080812;
}

.mic-badge  { right: -1px; }
.share-badge { left: -1px; }

.afk-badge {
  position: absolute;
  top: -3px;
  left: -3px;
  z-index: 5;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  line-height: 1;
  background: linear-gradient(145deg, #3a3a62, #222238);
  border: 1.5px solid #080812;
  box-shadow: 0 0 8px rgba(120, 140, 220, 0.35);
}

.mic-badge.active {
  background: #39ff14; color: #080812;
  box-shadow: 0 0 5px rgba(57,255,20,0.6);
}
.mic-badge.muted {
  background: #ff2957; color: #fff;
  box-shadow: 0 0 5px rgba(255,41,87,0.6);
}
.share-badge {
  background: #00f5ff; color: #080812;
  box-shadow: 0 0 5px rgba(0,245,255,0.6);
}

/* ─── Name row ───────────────────────────────────────────────────────────── */
.name-row {
  display: flex; align-items: center; gap: 5px;
  max-width: 100%; justify-content: center;
}

.name {
  font-weight: 600; font-size: 12px; color: #c8c8e8;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  max-width: 100px;
}

.you-tag {
  font-family: 'Orbitron', sans-serif;
  font-size: 7px; font-weight: 700; letter-spacing: 1px;
  color: #9d4edd; border: 1px solid #9d4edd;
  border-radius: 3px; padding: 1px 3px; flex-shrink: 0;
}

/* ─── Status ─────────────────────────────────────────────────────────────── */
.status-row { display: flex; justify-content: center; }

.status-chip {
  font-size: 10px; color: #a0a0c8;
  background: rgba(157,78,221,0.1);
  border: 1px solid rgba(157,78,221,0.25);
  border-radius: 10px;
  padding: 1px 7px;
  white-space: nowrap;
}

/* ─── Wave ───────────────────────────────────────────────────────────────── */
.wave-wrap {
  display: flex; align-items: flex-end; gap: 2px;
  height: 12px; opacity: 0; transition: opacity 0.2s;
}
.wave-wrap.visible { opacity: 1; }

.wave-bar {
  display: block; width: 3px; height: 4px;
  border-radius: 2px; background: #9d4edd;
  box-shadow: 0 0 4px #9d4edd;
  animation: wave 0.6s ease-in-out infinite alternate;
}
@keyframes wave {
  from { height: 3px; }
  to   { height: 11px; }
}

/* ─── Mobile ─────────────────────────────────────────────────────────────── */
@media (max-width: 640px) {
  .card {
    width: 112px;
    padding: 10px 8px 8px;
    border-radius: 9px;
  }
  .card-inner { gap: 5px; }

  .avatar-ring { width: 48px; height: 48px; }

  .name-row { font-size: 11px; }
  .name-text { max-width: 80px; }

  /* On mobile the overlay is triggered by tap — keep it full-card */
  .vol-overlay { gap: 9px; padding: 12px 10px; }
  .vol-label   { font-size: 9px; }
  .vol-pct     { font-size: 10px; }

  /* Small hint icon to indicate tap-for-volume */
  .card:not(.local)::after {
    content: '⋯';
    position: absolute;
    bottom: 4px;
    right: 6px;
    font-size: 10px;
    color: #3a3a5a;
    pointer-events: none;
  }
}
</style>
