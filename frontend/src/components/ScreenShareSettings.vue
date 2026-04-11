<template>
  <div class="ss-wrap" ref="wrapEl">
    <button
      class="gear-btn"
      :class="{ active: show }"
      title="Настройки трансляции"
      @click.stop="show = !show"
    >
      <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
        <path d="M19.14 12.94c.04-.3.06-.61.06-.94s-.02-.64-.07-.94l2.03-1.58c.18-.14.23-.41.12-.61l-1.92-3.32c-.12-.22-.37-.29-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54c-.04-.24-.24-.41-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.09.63-.09.94s.02.64.07.94l-2.03 1.58c-.18.14-.23.41-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"/>
      </svg>
    </button>

    <Transition name="popup">
      <div v-if="show" class="settings-popup" @click.stop>
        <div class="popup-title">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20 3H4c-1.1 0-2 .9-2 2v11c0 1.1.9 2 2 2h3l-1 1v1h12v-1l-1-1h3c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 13H4V5h16v11z"/>
          </svg>
          НАСТРОЙКИ ТРАНСЛЯЦИИ
        </div>

        <!-- Resolution -->
        <div class="setting-row">
          <span class="setting-label">Разрешение</span>
          <div class="chips">
            <button
              v-for="opt in RESOLUTIONS"
              :key="opt.value"
              class="chip"
              :class="{ active: settings.resolution === opt.value }"
              @click="settings.resolution = opt.value"
            >{{ opt.label }}</button>
          </div>
        </div>

        <!-- FPS -->
        <div class="setting-row">
          <span class="setting-label">Кадров/сек</span>
          <div class="chips">
            <button
              v-for="opt in FPS_OPTIONS"
              :key="opt.value"
              class="chip"
              :class="{ active: settings.fps === opt.value }"
              @click="settings.fps = opt.value"
            >{{ opt.label }}</button>
          </div>
        </div>

        <!-- Detail / effective bitrate (auto) -->
        <div class="setting-row">
          <span class="setting-label">Детализация</span>
          <div class="chips">
            <button
              v-for="opt in DETAIL_PRESETS"
              :key="opt.value"
              class="chip"
              :class="{ active: settings.detail === opt.value }"
              @click="settings.detail = opt.value"
            >{{ opt.label }}</button>
          </div>
        </div>

        <!-- Current effective values -->
        <div class="popup-summary">
          {{ summaryText }}
        </div>

        <div class="popup-note">
          Разрешение и FPS применяются при следующем запуске трансляции.<br/>
          Детализация и битрейт можно менять во время эфира — лимит считается автоматически с запасом.
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'

const RESOLUTIONS = [
  { value: 'auto',      label: 'Авто' },
  { value: '3840x2160', label: '4K' },
  { value: '1920x1080', label: '1080p' },
  { value: '1280x720',  label: '720p' },
  { value: '854x480',   label: '480p' },
  { value: '640x360',   label: '360p' },
]

const FPS_OPTIONS = [
  { value: 5,  label: '5' },
  { value: 15, label: '15' },
  { value: 30, label: '30' },
  { value: 60, label: '60' },
]

const DETAIL_PRESETS = [
  { value: 'low',       label: 'Меньше' },
  { value: 'balanced',  label: 'Баланс' },
  { value: 'high',      label: 'Больше' },
]

const props = defineProps({
  settings: { type: Object, required: true },
})

const show  = ref(false)
const wrapEl = ref(null)

const RES_LABELS = Object.fromEntries(RESOLUTIONS.map((r) => [r.value, r.label]))

function estimateBitrateLabel() {
  const { resolution, fps, detail: detRaw } = props.settings
  const detail = detRaw ?? 'balanced'
  let w = 1920
  let h = 1080
  if (resolution && resolution !== 'auto') {
    const parts = resolution.split('x').map(Number)
    if (parts.length === 2 && parts.every((n) => Number.isFinite(n) && n > 0)) [w, h] = parts
  }
  const pixels = w * h
  const detailMul = detail === 'low' ? 0.58 : detail === 'high' ? 1.42 : 1
  let kbps = pixels * fps * 0.00008 * detailMul * 1.15
  kbps = Math.round(Math.min(40000, Math.max(600, kbps)))
  return kbps >= 1000 ? `~${Math.round(kbps / 1000)} Мб/с` : `~${kbps} Кб/с`
}

const summaryText = computed(() => {
  const res = RES_LABELS[props.settings.resolution] ?? props.settings.resolution
  const det = DETAIL_PRESETS.find((d) => d.value === (props.settings.detail ?? 'balanced'))?.label ?? 'Баланс'
  return `${res} · ${props.settings.fps} fps · ${det} · ${estimateBitrateLabel()}`
})

function onOutside(e) {
  if (wrapEl.value && !wrapEl.value.contains(e.target)) show.value = false
}

onMounted(() => document.addEventListener('click', onOutside))
onBeforeUnmount(() => document.removeEventListener('click', onOutside))
</script>

<style scoped>
.ss-wrap { position: relative; }

/* ─── Gear button ────────────────────────────────────────────────────────── */
.gear-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px; height: 28px;
  border-radius: 6px;
  border: 1px solid #2e2e5f;
  background: rgba(30, 30, 63, 0.5);
  color: #7070a0;
  cursor: pointer;
  transition: all 0.15s;
  flex-shrink: 0;
}
.gear-btn:hover { border-color: #00f5ff; color: #00f5ff; background: rgba(0,245,255,0.07); }
.gear-btn.active {
  border-color: #9d4edd;
  color: #9d4edd;
  background: rgba(157,78,221,0.12);
  animation: spin-slow 4s linear infinite;
}

@keyframes spin-slow {
  from { transform: rotate(0deg); }
  to   { transform: rotate(360deg); }
}

/* ─── Popup ──────────────────────────────────────────────────────────────── */
.settings-popup {
  position: absolute;
  bottom: calc(100% + 10px);
  left: 50%;
  transform: translateX(-50%);
  width: 310px;
  background: rgba(8, 8, 20, 0.98);
  border: 1px solid #2e2e5f;
  border-radius: 12px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  z-index: 300;
  box-shadow: 0 8px 32px rgba(0,0,0,0.7), 0 0 0 1px rgba(157,78,221,0.1);
}

.popup-enter-active, .popup-leave-active {
  transition: opacity 0.15s, transform 0.15s;
}
.popup-enter-from, .popup-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(8px);
}

/* ─── Title ──────────────────────────────────────────────────────────────── */
.popup-title {
  display: flex;
  align-items: center;
  gap: 7px;
  font-family: 'Orbitron', sans-serif;
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 2px;
  color: #00f5ff;
  text-shadow: 0 0 8px rgba(0,245,255,0.4);
  padding-bottom: 10px;
  border-bottom: 1px solid #1e1e3f;
}

/* ─── Row ────────────────────────────────────────────────────────────────── */
.setting-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.setting-label {
  font-size: 10px;
  font-weight: 600;
  color: #7070a0;
  text-transform: uppercase;
  letter-spacing: 1px;
}

/* ─── Chips ──────────────────────────────────────────────────────────────── */
.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.chip {
  padding: 4px 10px;
  border-radius: 6px;
  border: 1px solid #2e2e5f;
  background: rgba(30, 30, 63, 0.5);
  color: #7070a0;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.12s;
  white-space: nowrap;
}
.chip:hover { border-color: #9d4edd; color: #c8a0f0; }
.chip.active {
  border-color: #9d4edd;
  background: rgba(157, 78, 221, 0.2);
  color: #c8a0f0;
  box-shadow: 0 0 8px rgba(157,78,221,0.3);
}

/* ─── Summary ────────────────────────────────────────────────────────────── */
.popup-summary {
  font-family: 'Orbitron', sans-serif;
  font-size: 10px;
  color: #9d4edd;
  text-align: center;
  padding: 6px 0 2px;
  letter-spacing: 1px;
}

/* ─── Note ───────────────────────────────────────────────────────────────── */
.popup-note {
  font-size: 10px;
  color: #50507a;
  text-align: center;
  line-height: 1.5;
  padding-top: 4px;
  border-top: 1px solid #1e1e3f;
}
</style>
