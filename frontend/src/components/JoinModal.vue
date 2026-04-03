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

      <div class="avatar-preview-wrap">
        <div class="avatar-ring" :class="{ 'has-avatar': avatarUrl }">
          <img
            v-if="avatarPreview"
            :src="avatarPreview"
            class="avatar-img"
            alt="avatar"
            @error="avatarPreview = null"
          />
          <div v-else class="avatar-placeholder">
            <svg width="40" height="40" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 12c2.7 0 4.8-2.1 4.8-4.8S14.7 2.4 12 2.4 7.2 4.5 7.2 7.2 9.3 12 12 12zm0 2.4c-3.2 0-9.6 1.6-9.6 4.8v2.4h19.2v-2.4c0-3.2-6.4-4.8-9.6-4.8z"/>
            </svg>
          </div>
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

        <div v-if="props.error" class="form-error">
          {{ props.error }}
        </div>

        <button type="submit" class="join-btn" :disabled="!name.trim()">
          <span class="btn-text">ВОЙТИ В КОМНАТУ</span>
          <span class="btn-arrow">▶</span>
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  error: { type: String, default: null },
})

const emit = defineEmits(['join'])

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

const savedName = getCookie('zvonok_name')
const savedAvatar = getCookie('zvonok_avatar')

const name = ref(savedName)
const avatarUrl = ref(savedAvatar)
const avatarPreview = ref(savedAvatar || null)

watch(avatarUrl, (val) => {
  avatarPreview.value = val.trim() || null
})

function handleSubmit() {
  if (!name.value.trim()) return
  setCookie('zvonok_name', name.value.trim())
  setCookie('zvonok_avatar', avatarUrl.value.trim())
  emit('join', {
    name: name.value.trim(),
    avatar: avatarUrl.value.trim(),
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
}

.modal {
  position: relative;
  width: 420px;
  max-width: calc(100vw - 32px);
  background: #0f0f1e;
  border: 1px solid #2e2e5f;
  border-radius: 12px;
  padding: 40px 36px;
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
  font-size: 36px;
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
}

.avatar-preview-wrap {
  display: flex;
  justify-content: center;
  margin-bottom: 24px;
}

.avatar-ring {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  border: 2px solid #1e1e3f;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: border-color 0.3s, box-shadow 0.3s;
  overflow: hidden;
  background: #12122a;
}

.avatar-ring.has-avatar {
  border-color: #9d4edd;
  box-shadow: 0 0 12px rgba(157, 78, 221, 0.5), 0 0 30px rgba(157, 78, 221, 0.2);
}

.avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-placeholder {
  color: #2e2e5f;
}

.modal-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
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
  color: #4040606;
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
  margin-top: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 13px 20px;
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
</style>
