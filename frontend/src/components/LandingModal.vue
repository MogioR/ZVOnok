<template>
  <div class="overlay">
    <div class="modal">
      <div class="modal-glow" />

      <div class="modal-header">
        <div class="modal-logo">
          <span class="accent">Z</span>VOnok
        </div>
        <p class="modal-sub">ВИДЕОКОНФЕРЕНЦИИ СЛЕДУЮЩЕГО УРОВНЯ</p>
      </div>

      <!-- Create room form -->
      <div v-if="view === 'create'" class="create-form">
        <div class="back-row">
          <button class="back-btn" @click="view = 'main'">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
              <path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z"/>
            </svg>
            Назад
          </button>
        </div>

        <div class="section-title">СОЗДАТЬ КОМНАТУ</div>

        <div class="field">
          <label class="field-label">ПАРОЛЬ <span class="optional">(необязательно)</span></label>
          <input
            v-model="newRoomPassword"
            class="field-input"
            type="password"
            placeholder="Оставьте пустым для открытой комнаты"
            maxlength="64"
          />
        </div>

        <div v-if="createError" class="form-error">{{ createError }}</div>

        <button class="action-btn primary" :disabled="creating" @click="handleCreate">
          <span class="btn-text">{{ creating ? 'СОЗДАНИЕ…' : 'СОЗДАТЬ И ВОЙТИ' }}</span>
          <span class="btn-arrow">▶</span>
        </button>
      </div>

      <!-- Main choice -->
      <div v-else class="choice-grid">
        <button class="choice-card" @click="joinDefault">
          <div class="choice-icon">🌐</div>
          <div class="choice-title">ГЛАВНЫЙ ЗАЛ</div>
          <div class="choice-desc">Общедоступная комната,<br>всегда доступна</div>
        </button>

        <button class="choice-card" @click="view = 'create'">
          <div class="choice-icon">🔒</div>
          <div class="choice-title">СОЗДАТЬ КОМНАТУ</div>
          <div class="choice-desc">Приватная ссылка,<br>опциональный пароль</div>
        </button>
      </div>

      <div class="or-row">
        <div class="or-line" />
        <span class="or-text">или введите ID комнаты</span>
        <div class="or-line" />
      </div>

      <form class="join-by-id" @submit.prevent="handleJoinById">
        <input
          v-model="roomIdInput"
          class="field-input"
          type="text"
          placeholder="abc123..."
          maxlength="64"
        />
        <button type="submit" class="action-btn secondary" :disabled="!roomIdInput.trim()">
          Войти
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const emit = defineEmits(['select-room'])

const view = ref('main')
const newRoomPassword = ref('')
const roomIdInput = ref('')
const creating = ref(false)
const createError = ref('')

function generateRoomId() {
  return Array.from(crypto.getRandomValues(new Uint8Array(5)))
    .map((b) => b.toString(36).padStart(2, '0'))
    .join('')
    .slice(0, 8)
}

function joinDefault() {
  emit('select-room', { roomId: 'default', password: '' })
}

async function handleCreate() {
  createError.value = ''
  creating.value = true
  try {
    const roomId = generateRoomId()
    const res = await fetch('/api/rooms/create', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ roomId, password: newRoomPassword.value.trim() }),
    })
    if (!res.ok) {
      const txt = await res.text()
      createError.value = txt || 'Ошибка создания комнаты'
      return
    }
    const data = await res.json()
    // Update URL so the room link is shareable
    history.replaceState(null, '', '#' + data.roomId)
    emit('select-room', { roomId: data.roomId, password: newRoomPassword.value.trim() })
  } catch (e) {
    createError.value = 'Не удалось создать комнату'
  } finally {
    creating.value = false
  }
}

function handleJoinById() {
  const id = roomIdInput.value.trim()
  if (!id) return
  history.replaceState(null, '', '#' + id)
  emit('select-room', { roomId: id, password: '' })
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
}

.modal {
  position: relative;
  width: 460px;
  max-width: calc(100vw - 32px);
  background: #0f0f1e;
  border: 1px solid #2e2e5f;
  border-radius: 12px;
  padding: 36px 32px;
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
  margin-bottom: 28px;
}

.modal-logo {
  font-family: 'Orbitron', sans-serif;
  font-size: 32px;
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

/* ─── Choice grid ─────────────────────────────────────────────────────────── */
.choice-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 20px;
}

.choice-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 20px 12px;
  background: rgba(30, 30, 63, 0.5);
  border: 1px solid #2e2e5f;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
  text-align: center;
}

.choice-card:hover {
  border-color: #9d4edd;
  background: rgba(157, 78, 221, 0.08);
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(157, 78, 221, 0.2);
}

.choice-icon { font-size: 28px; }

.choice-title {
  font-family: 'Orbitron', sans-serif;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 1.5px;
  color: #c8a0f0;
}

.choice-desc {
  font-size: 11px;
  color: #7070a0;
  line-height: 1.4;
}

/* ─── Or divider ──────────────────────────────────────────────────────────── */
.or-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}

.or-line {
  flex: 1;
  height: 1px;
  background: #1e1e3f;
}

.or-text {
  font-size: 11px;
  color: #50507a;
  white-space: nowrap;
}

/* ─── Join by ID ──────────────────────────────────────────────────────────── */
.join-by-id {
  display: flex;
  gap: 10px;
}

.join-by-id .field-input {
  flex: 1;
}

/* ─── Create form ─────────────────────────────────────────────────────────── */
.create-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.back-row { margin-bottom: -4px; }

.back-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: none;
  color: #7070a0;
  font-size: 12px;
  cursor: pointer;
  transition: color 0.15s;
  padding: 0;
}
.back-btn:hover { color: #c8c8e8; }

.section-title {
  font-family: 'Orbitron', sans-serif;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 2px;
  color: #9d4edd;
}

/* ─── Shared ──────────────────────────────────────────────────────────────── */
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
  width: 100%;
  box-sizing: border-box;
}

.field-input::placeholder { color: #3a3a5a; }

.field-input:focus {
  border-color: #9d4edd;
  box-shadow: 0 0 0 2px rgba(157, 78, 221, 0.15);
}

.form-error {
  background: rgba(255, 41, 87, 0.1);
  border: 1px solid #ff2957;
  border-radius: 6px;
  padding: 8px 12px;
  color: #ff2957;
  font-size: 13px;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 12px 18px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
  flex-shrink: 0;
}

.action-btn.primary {
  background: linear-gradient(135deg, #6b2fa0, #9d4edd);
  border: 1px solid #9d4edd;
}

.action-btn.primary:hover:not(:disabled) {
  box-shadow: 0 0 20px rgba(157, 78, 221, 0.5);
  transform: translateY(-1px);
}

.action-btn.secondary {
  background: rgba(0, 245, 255, 0.08);
  border: 1px solid #00f5ff;
  color: #00f5ff;
}

.action-btn.secondary:hover:not(:disabled) {
  background: rgba(0, 245, 255, 0.15);
}

.action-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn-text {
  font-family: 'Orbitron', sans-serif;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 1.5px;
  color: #fff;
}

.btn-arrow {
  color: rgba(255, 255, 255, 0.7);
  font-size: 12px;
}
</style>
