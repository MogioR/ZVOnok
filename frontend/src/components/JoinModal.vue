<template>
  <div class="overlay">
    <div class="modal">
      <div class="modal-glow" />

      <div class="modal-header">
        <div class="modal-logo">
          <span class="accent">Z</span>VOnok
        </div>
        <p v-if="roomId && roomId !== 'default'" class="room-label">
          <span class="room-icon">🔑</span>
          КОМНАТА: {{ roomId }}
        </p>
        <p v-else class="modal-sub">ВИДЕОКОНФЕРЕНЦИИ СЛЕДУЮЩЕГО УРОВНЯ</p>
      </div>

      <!-- Media preview -->
      <div class="preview-row">
        <!-- Microphone level -->
        <div class="preview-block mic-block">
          <div class="preview-label">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3z"/>
            </svg>
            МИКРОФОН
          </div>
          <div class="mic-bar-wrap">
            <div class="mic-bar-fill" :style="{ width: micLevel + '%' }" />
          </div>
          <div class="preview-status" :class="micOk ? 'ok' : 'err'">
            {{ micOk ? 'Работает' : (micError || 'Нет доступа') }}
          </div>
        </div>

        <!-- Camera preview -->
        <div class="preview-block cam-block">
          <div class="preview-label">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
              <path d="M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z"/>
            </svg>
            КАМЕРА
          </div>
          <div class="cam-wrap">
            <video
              v-if="camStream"
              ref="previewVideoEl"
              class="cam-preview"
              autoplay
              playsinline
              muted
            />
            <div v-else class="cam-placeholder">
              <svg width="28" height="28" viewBox="0 0 24 24" fill="currentColor" style="opacity:0.3">
                <path d="M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z"/>
              </svg>
            </div>
          </div>
          <button class="cam-toggle" :class="{ active: camStream }" @click="toggleCamera">
            {{ camStream ? 'Выкл. камеру' : 'Вкл. камеру' }}
          </button>
        </div>
      </div>

      <form class="modal-form" @submit.prevent="handleSubmit">
        <div class="field">
          <label class="field-label">ПОЗЫВНОЙ</label>
          <input
            v-model="name"
            class="field-input"
            type="text"
            placeholder="Введите ваш ник..."
            maxlength="32"
            required
            autofocus
          />
        </div>

        <div class="field">
          <label class="field-label">URL АВАТАРА <span class="optional">(необязательно)</span></label>
          <input
            v-model="avatarUrl"
            class="field-input"
            type="url"
            placeholder="https://example.com/avatar.png"
          />
        </div>

        <div v-if="needsPassword" class="field">
          <label class="field-label">ПАРОЛЬ КОМНАТЫ</label>
          <input
            v-model="password"
            class="field-input"
            type="password"
            placeholder="Введите пароль..."
            maxlength="64"
            required
          />
        </div>

        <div v-if="props.error" class="form-error">
          {{ props.error }}
        </div>

        <button type="submit" class="join-btn" :disabled="!name.trim() || (needsPassword && !password.trim())">
          <span class="btn-text">ВОЙТИ В КОМНАТУ</span>
          <span class="btn-arrow">▶</span>
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'

const props = defineProps({
  error:         { type: String,  default: null },
  roomId:        { type: String,  default: 'default' },
  needsPassword: { type: Boolean, default: false },
})

const emit = defineEmits(['join'])

// ─── Cookie helpers ────────────────────────────────────────────────────────
function getCookie(key) {
  const m = document.cookie.match(
    new RegExp('(?:^|; )' + key.replace(/[[\]{}()*+?.,\\^$|#\s]/g, '\\$&') + '=([^;]*)')
  )
  return m ? decodeURIComponent(m[1]) : ''
}

function setCookie(key, value) {
  const expires = new Date(Date.now() + 365 * 864e5).toUTCString()
  document.cookie = `${key}=${encodeURIComponent(value)}; expires=${expires}; path=/; SameSite=Lax`
}

const savedName   = getCookie('zvonok_name')
const savedAvatar = getCookie('zvonok_avatar')

const name      = ref(savedName)
const avatarUrl = ref(savedAvatar)
const password  = ref('')

// ─── Mic preview ───────────────────────────────────────────────────────────
const micLevel = ref(0)
const micOk    = ref(false)
const micError = ref('')

let micStream    = null
let micCtx       = null
let micAnalyser  = null
let micTimer     = null

async function startMicPreview() {
  try {
    micStream = await navigator.mediaDevices.getUserMedia({
      audio: { echoCancellation: true, noiseSuppression: true },
      video: false,
    })
    micOk.value = true
    micCtx = new AudioContext()
    micAnalyser = micCtx.createAnalyser()
    micAnalyser.fftSize = 256
    micCtx.createMediaStreamSource(micStream).connect(micAnalyser)
    const data = new Uint8Array(micAnalyser.frequencyBinCount)
    micTimer = setInterval(() => {
      micAnalyser.getByteFrequencyData(data)
      const avg = data.reduce((a, b) => a + b, 0) / data.length
      micLevel.value = Math.min(100, avg * 2.5)
    }, 80)
  } catch (e) {
    micError.value = 'Нет доступа к микрофону'
  }
}

function stopMicPreview() {
  if (micTimer) { clearInterval(micTimer); micTimer = null }
  if (micCtx) { micCtx.close(); micCtx = null }
  micAnalyser = null
  if (micStream) { micStream.getTracks().forEach((t) => t.stop()); micStream = null }
  micLevel.value = 0
}

// ─── Camera preview ────────────────────────────────────────────────────────
const camStream     = ref(null)
const previewVideoEl = ref(null)

async function toggleCamera() {
  if (camStream.value) {
    camStream.value.getTracks().forEach((t) => t.stop())
    camStream.value = null
    if (previewVideoEl.value) previewVideoEl.value.srcObject = null
  } else {
    try {
      const s = await navigator.mediaDevices.getUserMedia({ video: true, audio: false })
      camStream.value = s
      nextTick(() => {
        if (previewVideoEl.value) {
          previewVideoEl.value.srcObject = s
          previewVideoEl.value.play().catch(() => {})
        }
      })
    } catch (e) {
      console.warn('Camera denied:', e)
    }
  }
}

// ─── Lifecycle ─────────────────────────────────────────────────────────────
onMounted(() => {
  startMicPreview()
})

onBeforeUnmount(() => {
  stopMicPreview()
  if (camStream.value) {
    camStream.value.getTracks().forEach((t) => t.stop())
    camStream.value = null
  }
})

// ─── Submit ────────────────────────────────────────────────────────────────
function handleSubmit() {
  if (!name.value.trim()) return
  if (props.needsPassword && !password.value.trim()) return
  setCookie('zvonok_name', name.value.trim())
  setCookie('zvonok_avatar', avatarUrl.value.trim())
  emit('join', {
    name:     name.value.trim(),
    avatar:   avatarUrl.value.trim(),
    password: password.value.trim(),
  })
}
</script>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(8, 8, 18, 0.95);
  backdrop-filter: blur(12px);
  overflow-y: auto;
  padding: 20px;
}

.modal {
  position: relative;
  width: 480px;
  max-width: 100%;
  background: #0f0f1e;
  border: 1px solid #2e2e5f;
  border-radius: 12px;
  padding: 32px 32px 28px;
  overflow: hidden;
}

.modal-glow {
  position: absolute;
  top: -60px;
  left: 50%;
  transform: translateX(-50%);
  width: 300px;
  height: 200px;
  background: radial-gradient(ellipse, rgba(157, 78, 221, 0.25) 0%, transparent 70%);
  pointer-events: none;
}

.modal-header {
  text-align: center;
  margin-bottom: 20px;
}

.modal-logo {
  font-family: 'Orbitron', sans-serif;
  font-size: 30px;
  font-weight: 900;
  letter-spacing: 4px;
  color: #e8e8ff;
  text-shadow: 0 0 30px rgba(157, 78, 221, 0.5);
  margin-bottom: 6px;
}

.accent {
  color: #9d4edd;
  text-shadow: 0 0 15px #9d4edd, 0 0 40px rgba(157, 78, 221, 0.5);
}

.modal-sub {
  font-family: 'Orbitron', sans-serif;
  font-size: 9px;
  font-weight: 400;
  letter-spacing: 3px;
  color: #7070a0;
  margin: 0;
}

.room-label {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-family: 'Orbitron', sans-serif;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 1.5px;
  color: #9d4edd;
  margin: 0;
}

.room-icon { font-size: 13px; }

/* ─── Media preview ───────────────────────────────────────────────────────── */
.preview-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-bottom: 18px;
}

.preview-block {
  background: rgba(15, 15, 30, 0.8);
  border: 1px solid #1e1e3f;
  border-radius: 8px;
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.preview-label {
  display: flex;
  align-items: center;
  gap: 5px;
  font-family: 'Orbitron', sans-serif;
  font-size: 8px;
  font-weight: 700;
  letter-spacing: 1.5px;
  color: #7070a0;
}

/* Mic level bar */
.mic-bar-wrap {
  height: 5px;
  background: #1e1e3f;
  border-radius: 3px;
  overflow: hidden;
}

.mic-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #39ff14, #00f5ff);
  border-radius: 3px;
  transition: width 0.08s linear;
}

.preview-status {
  font-size: 11px;
  font-weight: 600;
}

.preview-status.ok  { color: #39ff14; }
.preview-status.err { color: #ff2957; }

/* Camera */
.cam-wrap {
  width: 100%;
  aspect-ratio: 4/3;
  background: #080812;
  border-radius: 5px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

.cam-preview {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transform: scaleX(-1); /* mirror */
}

.cam-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}

.cam-toggle {
  padding: 4px 8px;
  border-radius: 5px;
  border: 1px solid #2e2e5f;
  background: transparent;
  color: #7070a0;
  font-size: 10px;
  cursor: pointer;
  transition: all 0.15s;
  align-self: flex-start;
}

.cam-toggle:hover { border-color: #9d4edd; color: #c8a0f0; }
.cam-toggle.active { border-color: #39ff14; color: #39ff14; }

/* ─── Form ────────────────────────────────────────────────────────────────── */
.modal-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-label {
  font-family: 'Orbitron', sans-serif;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 2px;
  color: #7070a0;
}

.optional {
  font-family: 'Rajdhani', sans-serif;
  font-size: 11px;
  letter-spacing: 0;
  text-transform: none;
  color: #404060;
}

.field-input {
  background: #080812;
  border: 1px solid #1e1e3f;
  border-radius: 6px;
  padding: 10px 14px;
  color: #c8c8e8;
  font-family: 'Rajdhani', sans-serif;
  font-size: 15px;
  font-weight: 500;
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.field-input::placeholder { color: #3a3a5a; }

.field-input:focus {
  border-color: #9d4edd;
  box-shadow: 0 0 0 2px rgba(157, 78, 221, 0.15), 0 0 12px rgba(157, 78, 221, 0.1);
}

.form-error {
  background: rgba(255, 41, 87, 0.1);
  border: 1px solid #ff2957;
  border-radius: 6px;
  padding: 8px 12px;
  color: #ff2957;
  font-size: 13px;
}

.join-btn {
  margin-top: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 12px 20px;
  background: linear-gradient(135deg, #6b2fa0, #9d4edd);
  border: 1px solid #9d4edd;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
  overflow: hidden;
}

.join-btn::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, rgba(255,255,255,0.05), transparent);
  opacity: 0;
  transition: opacity 0.2s;
}

.join-btn:hover:not(:disabled)::before { opacity: 1; }

.join-btn:hover:not(:disabled) {
  box-shadow: 0 0 20px rgba(157, 78, 221, 0.5), 0 0 40px rgba(157, 78, 221, 0.2);
  transform: translateY(-1px);
}

.join-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn-text {
  font-family: 'Orbitron', sans-serif;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 2px;
  color: #fff;
}

.btn-arrow {
  color: rgba(255,255,255,0.7);
  font-size: 12px;
  transition: transform 0.2s;
}

.join-btn:hover:not(:disabled) .btn-arrow {
  transform: translateX(3px);
}

@media (max-width: 480px) {
  .modal { padding: 24px 18px 20px; }
  .preview-row { grid-template-columns: 1fr; }
  .cam-wrap { aspect-ratio: 16/9; }
}
</style>
