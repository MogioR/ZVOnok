<template>
  <Teleport to="body">
    <Transition name="chat-slide">
      <div v-if="open" class="chat-panel">
        <div class="chat-header">
          <span class="chat-title">Чат</span>
          <button class="chat-close" @click="$emit('close')" title="Закрыть">✕</button>
        </div>

        <div class="messages-list" ref="messagesEl">
          <div v-if="messages.length === 0" class="messages-empty">
            Нет сообщений. Напиши первым!
          </div>

          <div
            v-for="msg in messages"
            :key="msg.id"
            class="message"
            :class="{ 'message-local': msg.isLocal }"
          >
            <img
              v-if="!msg.isLocal && msg.avatar"
              :src="msg.avatar"
              class="msg-avatar"
              @error="e => e.target.style.display='none'"
              alt=""
            />
            <div v-else-if="!msg.isLocal" class="msg-avatar-fallback">
              {{ (msg.name || '?')[0].toUpperCase() }}
            </div>

            <div class="msg-body">
              <div v-if="!msg.isLocal" class="msg-name">{{ msg.name }}</div>

              <!-- Text with inline media detection -->
              <div class="msg-bubble">
                <template v-for="(part, i) in parsedParts(msg.text)" :key="i">
                  <img
                    v-if="part.type === 'image'"
                    :src="part.value"
                    class="msg-media"
                    loading="lazy"
                    @error="e => e.target.style.display='none'"
                    alt="media"
                  />
                  <a
                    v-else-if="part.type === 'link'"
                    :href="part.value"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="msg-link"
                  >{{ part.value }}</a>
                  <span v-else>{{ part.value }}</span>
                </template>
              </div>

              <div class="msg-time">{{ formatTime(msg.timestamp) }}</div>
            </div>
          </div>
        </div>

        <div class="chat-input-area">
          <!-- Emoji picker -->
          <div class="emoji-wrap">
            <button class="emoji-btn" @click="showEmoji = !showEmoji" title="Смайлики">
              😊
            </button>
            <div v-if="showEmoji" class="emoji-picker">
              <button
                v-for="em in EMOJIS"
                :key="em"
                class="emoji-item"
                @click="insertEmoji(em)"
              >{{ em }}</button>
            </div>
          </div>

          <textarea
            ref="inputEl"
            v-model="draft"
            class="chat-textarea"
            placeholder="Напиши сообщение…"
            rows="1"
            @keydown="onKeydown"
            @input="autoResize"
          />

          <button
            class="send-btn"
            :disabled="!draft.trim()"
            @click="sendMessage"
            title="Отправить (Enter)"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
              <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/>
            </svg>
          </button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'

const props = defineProps({
  open:     { type: Boolean, default: false },
  messages: { type: Array, default: () => [] },
})

const emit = defineEmits(['close', 'send'])

const EMOJIS = [
  '😀','😂','😍','🥰','😎','😭','😤','🙄','😴','🤔',
  '💀','🔥','❤️','✅','👍','👎','🎉','🙏','💪','🤯',
  '😱','🤣','😅','😊','🎌','📞','☕','🎯','🍕','🐱',
]

const IMAGE_EXTS = /\.(gif|jpg|jpeg|png|webp|avif)(\?[^\s]*)?$/i
const GIPHY_RE   = /(?:giphy\.com|media\d*\.giphy\.com|tenor\.com)/i
const URL_RE     = /(https?:\/\/[^\s]+)/g

function parsedParts(text) {
  const parts = []
  let last = 0
  let m
  URL_RE.lastIndex = 0
  while ((m = URL_RE.exec(text)) !== null) {
    if (m.index > last) parts.push({ type: 'text', value: text.slice(last, m.index) })
    const url = m[0]
    if (IMAGE_EXTS.test(url) || GIPHY_RE.test(url)) {
      parts.push({ type: 'image', value: url })
    } else {
      parts.push({ type: 'link', value: url })
    }
    last = m.index + url.length
  }
  if (last < text.length) parts.push({ type: 'text', value: text.slice(last) })
  return parts
}

function formatTime(ts) {
  return new Date(ts).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

const draft    = ref('')
const showEmoji = ref(false)
const inputEl  = ref(null)
const messagesEl = ref(null)

function autoResize() {
  const el = inputEl.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 120) + 'px'
}

function onKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}

function sendMessage() {
  if (!draft.value.trim()) return
  emit('send', draft.value)
  draft.value = ''
  nextTick(() => {
    if (inputEl.value) { inputEl.value.style.height = 'auto' }
  })
  showEmoji.value = false
}

function insertEmoji(em) {
  const el = inputEl.value
  if (!el) { draft.value += em; return }
  const s = el.selectionStart
  const e2 = el.selectionEnd
  draft.value = draft.value.slice(0, s) + em + draft.value.slice(e2)
  nextTick(() => {
    el.setSelectionRange(s + em.length, s + em.length)
    el.focus()
  })
}

function scrollToBottom(behavior = 'smooth') {
  nextTick(() => {
    if (messagesEl.value) {
      messagesEl.value.scrollTo({ top: messagesEl.value.scrollHeight, behavior })
    }
  })
}

// Scroll to bottom on new messages
watch(() => props.messages.length, () => scrollToBottom('smooth'))

// When panel opens: scroll instantly to bottom, close emoji picker
watch(() => props.open, (v) => {
  if (v) scrollToBottom('instant')
  else showEmoji.value = false
})
</script>

<style scoped>
.chat-panel {
  position: fixed;
  top: var(--header-h, 56px);
  right: 0;
  bottom: var(--bar-h, 72px);
  width: 320px;
  background: rgba(9, 9, 20, 0.97);
  border-left: 1px solid #1e1e3f;
  display: flex;
  flex-direction: column;
  z-index: 50;
  backdrop-filter: blur(12px);
}

/* ─── Slide transition ───────────────────────────────────────────────────── */
.chat-slide-enter-active,
.chat-slide-leave-active {
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
.chat-slide-enter-from,
.chat-slide-leave-to {
  transform: translateX(100%);
}

/* ─── Header ─────────────────────────────────────────────────────────────── */
.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  height: 44px;
  border-bottom: 1px solid #1e1e3f;
  flex-shrink: 0;
}

.chat-title {
  font-family: 'Orbitron', sans-serif;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 2px;
  color: #c8c8e8;
}

.chat-close {
  background: none;
  border: none;
  color: #7070a0;
  font-size: 14px;
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 4px;
  transition: color 0.15s, background 0.15s;
}
.chat-close:hover { color: #ff2957; background: rgba(255,41,87,0.1); }

/* ─── Messages ───────────────────────────────────────────────────────────── */
.messages-list {
  flex: 1;
  overflow-y: auto;
  padding: 12px 10px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  scrollbar-width: thin;
  scrollbar-color: #2e2e5f transparent;
}
.messages-list::-webkit-scrollbar { width: 4px; }
.messages-list::-webkit-scrollbar-thumb { background: #2e2e5f; border-radius: 2px; }

.messages-empty {
  text-align: center;
  color: #7070a0;
  font-size: 13px;
  margin-top: 24px;
}

/* ─── Single message ─────────────────────────────────────────────────────── */
.message {
  display: flex;
  gap: 8px;
  align-items: flex-end;
}

.message-local {
  flex-direction: row-reverse;
}

.msg-avatar {
  width: 28px; height: 28px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}
.msg-avatar-fallback {
  width: 28px; height: 28px;
  border-radius: 50%;
  background: #1e1e3f;
  color: #9d4edd;
  display: flex; align-items: center; justify-content: center;
  font-size: 13px; font-weight: 700;
  flex-shrink: 0;
}

.msg-body {
  max-width: 70%;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.message-local .msg-body { align-items: flex-end; }

.msg-name {
  font-size: 10px;
  color: #7070a0;
  padding: 0 4px;
}

.msg-bubble {
  background: #1a1a35;
  border: 1px solid #2e2e5f;
  border-radius: 10px 10px 10px 2px;
  padding: 7px 10px;
  font-size: 13px;
  color: #c8c8e8;
  line-height: 1.45;
  word-break: break-word;
  white-space: pre-wrap;
}

.message-local .msg-bubble {
  background: rgba(157, 78, 221, 0.15);
  border-color: rgba(157, 78, 221, 0.4);
  border-radius: 10px 10px 2px 10px;
}

.msg-media {
  max-width: 200px;
  max-height: 200px;
  border-radius: 8px;
  display: block;
  margin-top: 4px;
  object-fit: contain;
}

.msg-link {
  color: #00f5ff;
  text-decoration: underline;
  word-break: break-all;
}

.msg-time {
  font-size: 9px;
  color: #50507a;
  padding: 0 4px;
}

/* ─── Input area ─────────────────────────────────────────────────────────── */
.chat-input-area {
  flex-shrink: 0;
  display: flex;
  align-items: flex-end;
  gap: 6px;
  padding: 8px 10px;
  border-top: 1px solid #1e1e3f;
}

.emoji-wrap { position: relative; flex-shrink: 0; }

.emoji-btn {
  background: none;
  border: none;
  font-size: 18px;
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  line-height: 1;
  transition: transform 0.15s;
}
.emoji-btn:hover { transform: scale(1.2); }

.emoji-picker {
  position: absolute;
  bottom: calc(100% + 6px);
  left: 0;
  width: 220px;
  background: rgba(10, 10, 26, 0.98);
  border: 1px solid #2e2e5f;
  border-radius: 10px;
  padding: 8px;
  display: flex;
  flex-wrap: wrap;
  gap: 2px;
  z-index: 200;
  box-shadow: 0 8px 24px rgba(0,0,0,0.6);
}

.emoji-item {
  background: none;
  border: none;
  font-size: 18px;
  cursor: pointer;
  padding: 4px 5px;
  border-radius: 6px;
  transition: background 0.1s;
  line-height: 1;
}
.emoji-item:hover { background: rgba(157,78,221,0.2); }

.chat-textarea {
  flex: 1;
  background: #0f0f1e;
  border: 1px solid #2e2e5f;
  border-radius: 8px;
  color: #c8c8e8;
  font-size: 13px;
  padding: 8px 10px;
  resize: none;
  outline: none;
  line-height: 1.45;
  max-height: 120px;
  overflow-y: auto;
  font-family: inherit;
  transition: border-color 0.15s;
  scrollbar-width: thin;
  scrollbar-color: #2e2e5f transparent;
}
.chat-textarea:focus { border-color: #9d4edd; }
.chat-textarea::placeholder { color: #50507a; }

.send-btn {
  flex-shrink: 0;
  width: 34px; height: 34px;
  border-radius: 8px;
  border: none;
  background: rgba(157,78,221,0.2);
  color: #9d4edd;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer;
  transition: background 0.15s, transform 0.1s;
}
.send-btn:hover:not(:disabled) {
  background: rgba(157,78,221,0.4);
  transform: scale(1.05);
}
.send-btn:disabled { opacity: 0.4; cursor: not-allowed; }

/* ─── Mobile: full-width slide-over ──────────────────────────────────────── */
@media (max-width: 640px) {
  .chat-panel {
    top: var(--header-h, 48px);
    bottom: var(--bar-h, 58px);
    width: 100vw;
    border-left: none;
    border-top: 1px solid #1e1e3f;
  }
}
</style>
