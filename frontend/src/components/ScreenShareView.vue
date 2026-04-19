<template>
  <div class="screen-view">
    <!-- ─── Header ──────────────────────────────────────────────────────────── -->
    <div class="screen-header">
      <div class="screen-title">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
          <path d="M20 3H4c-1.1 0-2 .9-2 2v11c0 1.1.9 2 2 2h3l-1 1v1h12v-1l-1-1h3c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 13H4V5h16v11z"/>
        </svg>
        ТРАНСЛЯЦИЯ ЭКРАНА
      </div>

      <!-- Webcam toggle button (always visible when in stream view) -->
      <button
        class="webcam-btn"
        :class="{ active: isWebcamActive }"
        :title="isWebcamActive ? 'Выключить веб-камеру' : 'Включить веб-камеру'"
        @click="$emit('toggle-webcam')"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
          <path v-if="!isWebcamActive" d="M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z"/>
          <path v-else d="M21 6.5l-4 4V7c0-.55-.45-1-1-1H9.82L21 17.18V6.5zM3.27 2L2 3.27 4.73 6H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.21 0 .39-.08.54-.18L19.73 21 21 19.73 3.27 2z"/>
        </svg>
        <span class="webcam-label">{{ isWebcamActive ? 'Камера вкл.' : 'Камера' }}</span>
      </button>

      <!-- View mode toggle (only with 2+ sharers) -->
      <div v-if="sharers.length > 1" class="mode-btns">
        <button
          class="mode-btn"
          :class="{ active: mode === 'single' }"
          title="Один поток"
          @click="mode = 'single'"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
            <path d="M21 3H3c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h18c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H3V5h18v14z"/>
          </svg>
        </button>
        <button
          class="mode-btn"
          :class="{ active: mode === 'split' }"
          title="Разделить экраны"
          @click="mode = 'split'"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
            <path d="M3 3h8v8H3zm0 10h8v8H3zM13 3h8v8h-8zm0 10h8v8h-8z"/>
          </svg>
        </button>
      </div>

      <!-- Sharer tabs (single mode + multiple sharers) -->
      <div v-if="mode === 'single' && sharers.length > 1" class="sharer-tabs">
        <button
          v-for="s in sharers"
          :key="s.id"
          class="sharer-tab"
          :class="{ active: resolvedActiveId === s.id }"
          @click="$emit('select', s.id)"
        >
          <img
            v-if="s.avatar"
            :src="s.avatar"
            class="tab-avatar"
            alt=""
            @error="$event.target.style.display='none'"
          />
          <span>{{ s.name }}</span>
        </button>
      </div>

      <!-- Active name (single mode + 1 sharer) -->
      <div v-else-if="mode === 'single' && activeName" class="active-sharer-name">
        {{ activeName }}
      </div>

      <!-- Volume (single mode only) -->
      <div v-if="mode === 'single'" class="screen-vol">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
          <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/>
        </svg>
        <input
          type="range"
          min="0" max="1" step="0.01"
          :value="singleVol"
          class="vol-slider"
          title="Громкость стрима"
          @input="singleVol = +$event.target.value; applySingleVol()"
        />
        <span class="vol-pct">{{ Math.round(singleVol * 100) }}%</span>
      </div>
    </div>

    <!-- ─── Single mode video ───────────────────────────────────────────────── -->
    <div v-if="mode === 'single'" class="video-wrap">
      <video
        ref="singleVideoEl"
        class="screen-video"
        autoplay
        playsinline
        :muted="isLocalActive || hasScreenAudio || deafened"
      />
      <div v-if="!activeStream" class="no-stream">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="currentColor" style="opacity:0.3">
          <path d="M20 3H4c-1.1 0-2 .9-2 2v11c0 1.1.9 2 2 2h3l-1 1v1h12v-1l-1-1h3c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 13H4V5h16v11z"/>
        </svg>
        <span>Ожидание трансляции...</span>
      </div>

      <!-- Webcam PIP overlay (local webcam or active sharer's webcam) -->
      <div v-if="pipStream" class="webcam-pip">
        <video
          ref="pipVideoEl"
          class="pip-video"
          autoplay
          playsinline
          muted
        />
      </div>
    </div>

    <!-- ─── Split mode grid ────────────────────────────────────────────────── -->
    <div v-else class="split-grid" :style="{ '--cols': splitCols }">
      <div
        v-for="s in sharers"
        :key="s.id"
        class="split-cell"
      >
        <video
          :ref="el => setSplitRef(s.id, el)"
          class="split-video"
          autoplay
          playsinline
          :muted="s.id === localId || s.hasScreenAudio || deafened"
        />
        <div class="cell-footer">
          <div class="cell-name">
            <img
              v-if="s.avatar"
              :src="s.avatar"
              class="cell-avatar"
              alt=""
              @error="$event.target.style.display='none'"
            />
            <span>{{ s.name }}</span>
          </div>
          <div v-if="s.id !== localId" class="cell-vol">
            <svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor">
              <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02z"/>
            </svg>
            <input
              type="range"
              min="0" max="1" step="0.01"
              :value="splitVols[s.id] ?? 1"
              class="cell-vol-slider"
              @input="setSplitVol(s.id, +$event.target.value)"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'

const props = defineProps({
  sharers:       { type: Array,   required: true },
  activeId:      { type: String,  default: null },
  localId:       { type: String,  default: null },
  deafened:      { type: Boolean, default: false },
  /** Local webcam stream — shown as PIP when active */
  webcamStream:  { type: Object,  default: null },
  isWebcamActive: { type: Boolean, default: false },
})

const emit = defineEmits(['select', 'volume-change', 'screen-volume-change', 'toggle-webcam'])

// ─── View mode ─────────────────────────────────────────────────────────────
const mode = ref('single')

watch(() => props.sharers.length, (n) => {
  if (n <= 1) mode.value = 'single'
})

// ─── Single mode ───────────────────────────────────────────────────────────
const singleVideoEl = ref(null)
const singleVol     = ref(1.0)
const pipVideoEl    = ref(null)

const resolvedActiveId = computed(() => {
  if (!props.sharers.length) return null
  return props.activeId ?? props.sharers[0]?.id ?? null
})

const activeSharer  = computed(() =>
  props.sharers.find((s) => s.id === resolvedActiveId.value) ?? null
)

const activeName    = computed(() => activeSharer.value?.name ?? null)
const activeStream  = computed(() => activeSharer.value?.screenStream ?? null)
const isLocalActive = computed(() => activeSharer.value?.id === props.localId)
const hasScreenAudio = computed(() => !!activeSharer.value?.hasScreenAudio)

// Show local webcam as PIP, OR the active sharer's webcam if they have one
const pipStream = computed(() => {
  if (props.webcamStream) return props.webcamStream
  if (activeSharer.value?.webcamStream) return activeSharer.value.webcamStream
  return null
})

watch(
  activeStream,
  (stream) => {
    nextTick(() => {
      if (!singleVideoEl.value) return
      singleVideoEl.value.srcObject = stream ?? null
      singleVideoEl.value.volume = singleVol.value
      if (stream) singleVideoEl.value.play().catch(() => {})
    })
  },
  { immediate: true },
)

watch(
  () => [resolvedActiveId.value, activeSharer.value?.screenVolume],
  () => {
    const v = activeSharer.value?.screenVolume
    if (typeof v === 'number') singleVol.value = v
  },
)

// Attach PIP stream
watch(
  pipStream,
  (stream) => {
    nextTick(() => {
      if (!pipVideoEl.value) return
      pipVideoEl.value.srcObject = stream ?? null
      if (stream) pipVideoEl.value.play().catch(() => {})
    })
  },
  { immediate: true },
)

function applySingleVol() {
  if (singleVideoEl.value) singleVideoEl.value.volume = singleVol.value
  if (resolvedActiveId.value) emit('screen-volume-change', resolvedActiveId.value, singleVol.value)
}

// ─── Split mode ────────────────────────────────────────────────────────────
const splitRefs = new Map()
const splitVols = ref({})

const splitCols = computed(() => {
  const n = props.sharers.length
  if (n <= 1) return 1
  if (n <= 4) return 2
  return 3
})

function setSplitRef(id, el) {
  if (el) {
    splitRefs.set(id, el)
    const sharer = props.sharers.find((s) => s.id === id)
    if (sharer?.screenStream) {
      el.srcObject = sharer.screenStream
      el.volume = splitVols.value[id] ?? 1
      el.play().catch(() => {})
    }
  } else {
    splitRefs.delete(id)
  }
}

watch(
  () => props.sharers.map((s) => s.id + '|' + (s.screenStream ? '1' : '0')),
  () => {
    nextTick(() => {
      props.sharers.forEach((s) => {
        const el = splitRefs.get(s.id)
        if (!el) return
        if (el.srcObject !== s.screenStream) {
          el.srcObject = s.screenStream ?? null
          el.volume = splitVols.value[s.id] ?? 1
          if (s.screenStream) el.play().catch(() => {})
        }
      })
    })
  },
)

watch(mode, (m) => {
  if (m === 'split') {
    nextTick(() => {
      props.sharers.forEach((s) => {
        const el = splitRefs.get(s.id)
        if (el && s.screenStream && el.srcObject !== s.screenStream) {
          el.srcObject = s.screenStream
          el.volume = splitVols.value[s.id] ?? 1
          el.play().catch(() => {})
        }
      })
    })
  }
})

function setSplitVol(id, val) {
  splitVols.value = { ...splitVols.value, [id]: val }
  const el = splitRefs.get(id)
  if (el) el.volume = val
  emit('screen-volume-change', id, val)
}
</script>

<style scoped>
.screen-view {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  border: 1px solid #1e1e3f;
  border-radius: 10px;
  overflow: hidden;
  background: #0a0a1a;
  box-shadow: 0 0 20px rgba(0, 245, 255, 0.06), inset 0 0 40px rgba(0, 0, 0, 0.3);
  width: 100%;
  height: 100%;
}

/* ─── Header ─────────────────────────────────────────────────────────────── */
.screen-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  background: rgba(0, 245, 255, 0.04);
  border-bottom: 1px solid #1e1e3f;
  flex-shrink: 0;
  flex-wrap: wrap;
}

.screen-title {
  display: flex;
  align-items: center;
  gap: 7px;
  font-family: 'Orbitron', sans-serif;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 2px;
  color: #00f5ff;
  text-shadow: 0 0 8px rgba(0, 245, 255, 0.4);
  white-space: nowrap;
}

/* ─── Webcam button ──────────────────────────────────────────────────────── */
.webcam-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  border: 1px solid #2e2e5f;
  border-radius: 6px;
  background: transparent;
  color: #7070a0;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
  flex-shrink: 0;
}

.webcam-btn:hover {
  border-color: #9d4edd;
  color: #c8a0f0;
  background: rgba(157, 78, 221, 0.08);
}

.webcam-btn.active {
  border-color: #39ff14;
  color: #39ff14;
  background: rgba(57, 255, 20, 0.08);
  box-shadow: 0 0 8px rgba(57, 255, 20, 0.2);
}

.webcam-label {
  font-family: 'Rajdhani', sans-serif;
}

/* ─── Mode toggle ────────────────────────────────────────────────────────── */
.mode-btns {
  display: flex;
  gap: 3px;
  flex-shrink: 0;
  background: rgba(30, 30, 63, 0.5);
  border: 1px solid #2e2e5f;
  border-radius: 6px;
  padding: 2px;
}

.mode-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px; height: 22px;
  border: none; border-radius: 4px;
  background: transparent; color: #50507a;
  cursor: pointer; transition: all 0.15s;
}
.mode-btn:hover { color: #c8c8e8; }
.mode-btn.active {
  background: rgba(0, 245, 255, 0.12);
  color: #00f5ff;
  box-shadow: 0 0 6px rgba(0, 245, 255, 0.2);
}

/* ─── Sharer tabs ────────────────────────────────────────────────────────── */
.sharer-tabs {
  display: flex;
  gap: 6px;
  overflow-x: auto;
  flex: 1;
}
.sharer-tabs::-webkit-scrollbar { height: 3px; }
.sharer-tabs::-webkit-scrollbar-thumb { background: #1e1e3f; border-radius: 2px; }

.sharer-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: transparent;
  border: 1px solid #1e1e3f;
  border-radius: 4px;
  cursor: pointer;
  color: #7070a0;
  font-family: 'Rajdhani', sans-serif;
  font-size: 13px; font-weight: 600;
  white-space: nowrap; transition: all 0.2s;
}
.sharer-tab:hover { border-color: #00f5ff; color: #00f5ff; }
.sharer-tab.active {
  border-color: #00f5ff; color: #00f5ff;
  background: rgba(0, 245, 255, 0.08);
  box-shadow: 0 0 8px rgba(0, 245, 255, 0.2);
}

.tab-avatar {
  width: 18px; height: 18px;
  border-radius: 50%; object-fit: cover;
}

.active-sharer-name {
  font-weight: 600; font-size: 14px;
  color: #c8c8e8; flex: 1;
}

/* ─── Volume (single) ────────────────────────────────────────────────────── */
.screen-vol {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
  color: #7070a0; flex-shrink: 0;
}

.vol-slider {
  width: 80px;
  -webkit-appearance: none;
  height: 3px; border-radius: 2px;
  background: #1e1e3f;
  outline: none; cursor: pointer;
}
.vol-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  width: 10px; height: 10px; border-radius: 50%;
  background: #00f5ff;
  box-shadow: 0 0 4px rgba(0, 245, 255, 0.6);
  cursor: pointer;
}
.vol-slider::-moz-range-thumb {
  width: 10px; height: 10px; border-radius: 50%;
  background: #00f5ff; border: none; cursor: pointer;
}

.vol-pct {
  font-family: 'Orbitron', sans-serif;
  font-size: 9px; color: #7070a0;
  width: 30px; text-align: right;
}

/* ─── Single video ───────────────────────────────────────────────────────── */
.video-wrap {
  flex: 1;
  min-height: 0;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #050510;
}

.screen-video {
  width: 100%; height: 100%;
  object-fit: contain;
  display: block;
}

.no-stream {
  position: absolute; inset: 0;
  display: flex; flex-direction: column;
  align-items: center; justify-content: center;
  gap: 12px; color: #7070a0;
  font-size: 14px; font-weight: 500;
}

/* ─── Webcam PIP ─────────────────────────────────────────────────────────── */
.webcam-pip {
  position: absolute;
  bottom: 14px;
  right: 14px;
  width: 160px;
  aspect-ratio: 4/3;
  border-radius: 8px;
  overflow: hidden;
  border: 2px solid rgba(57, 255, 20, 0.6);
  box-shadow: 0 4px 20px rgba(0,0,0,0.6), 0 0 12px rgba(57,255,20,0.25);
  background: #050510;
  z-index: 10;
}

.pip-video {
  width: 100%; height: 100%;
  object-fit: cover;
  transform: scaleX(-1); /* mirror for local camera feel */
  display: block;
}

/* ─── Split grid ─────────────────────────────────────────────────────────── */
.split-grid {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: repeat(var(--cols, 2), 1fr);
  gap: 3px; padding: 3px;
  background: #080810; overflow: hidden;
}

.split-cell {
  position: relative;
  background: #050510;
  border-radius: 6px; overflow: hidden;
  border: 1px solid #1a1a3a;
  min-height: 0;
}

.split-video {
  width: 100%; height: 100%;
  object-fit: contain; display: block;
}

.cell-footer {
  position: absolute;
  bottom: 0; left: 0; right: 0;
  display: flex; align-items: center;
  justify-content: space-between;
  padding: 5px 8px;
  background: linear-gradient(transparent, rgba(5, 5, 16, 0.85));
  gap: 8px;
  opacity: 0; transition: opacity 0.2s;
}
.split-cell:hover .cell-footer { opacity: 1; }

.cell-name {
  display: flex; align-items: center; gap: 5px;
  font-size: 11px; font-weight: 600; color: #c8c8e8;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  text-shadow: 0 1px 3px rgba(0,0,0,0.8);
}

.cell-avatar {
  width: 16px; height: 16px;
  border-radius: 50%; object-fit: cover; flex-shrink: 0;
}

.cell-vol {
  display: flex; align-items: center; gap: 4px;
  color: #7070a0; flex-shrink: 0;
}

.cell-vol-slider {
  width: 60px;
  -webkit-appearance: none;
  height: 3px; border-radius: 2px;
  background: rgba(255,255,255,0.2);
  outline: none; cursor: pointer;
}
.cell-vol-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  width: 9px; height: 9px; border-radius: 50%;
  background: #00f5ff; cursor: pointer;
}
.cell-vol-slider::-moz-range-thumb {
  width: 9px; height: 9px; border-radius: 50%;
  background: #00f5ff; border: none; cursor: pointer;
}

@media (max-width: 640px) {
  .webcam-pip {
    width: 100px;
    bottom: 8px;
    right: 8px;
  }
  .webcam-label { display: none; }
}
</style>
