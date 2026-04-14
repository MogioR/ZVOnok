import { reactive, computed, shallowRef, ref, markRaw, watch } from 'vue'

// ─── Chat persistence (localStorage, 24h TTL) ─────────────────────────────────

const CHAT_KEY    = 'zvonok_chat_v1'
const CHAT_MAX_MS = 24 * 60 * 60 * 1000

function loadChatHistory() {
  try {
    const raw = localStorage.getItem(CHAT_KEY)
    if (!raw) return []
    const msgs = JSON.parse(raw)
    const cutoff = Date.now() - CHAT_MAX_MS
    return Array.isArray(msgs) ? msgs.filter((m) => m.timestamp > cutoff) : []
  } catch {
    return []
  }
}

function saveChatHistory(msgs) {
  try {
    const cutoff = Date.now() - CHAT_MAX_MS
    localStorage.setItem(CHAT_KEY, JSON.stringify(msgs.filter((m) => m.timestamp > cutoff)))
  } catch {}
}

// ─── ICE Server Config ────────────────────────────────────────────────────────

let iceServersPromise = null
function fetchIceServers() {
  if (iceServersPromise) return iceServersPromise
  iceServersPromise = fetch('/ice-servers')
    .then((r) => r.json())
    .then((servers) => ({ iceServers: servers }))
    .catch(() => ({
      iceServers: [
        { urls: 'stun:stun.l.google.com:19302' },
        { urls: 'stun:stun1.l.google.com:19302' },
      ],
    }))
  return iceServersPromise
}

/** Звук экрана/вкладки может прийти до видео; не перезаписывать им микрофон в audioEl. */
function likelyDisplayCaptureAudio(track) {
  if (!track || track.kind !== 'audio') return false
  const L = (track.label || '').toLowerCase()
  if (/(microphone|микрофон|\bmic\b|headset|headphone|headphones|earphone|наушники|встроенный|built-in audio|array microphone)/.test(L)) {
    return false
  }
  if (/(display|screen|window|system|desktop|tab|capture|share|stereo mix|что воспроизводится|вкладка)/.test(L)) {
    return true
  }
  return false
}

export function useConference() {
  const myId = shallowRef(null)
  const myInfo = shallowRef(null)
  const myStatus = ref(null)

  const participants = reactive(new Map())
  const peerMeta = {}

  const localStream = shallowRef(null)
  const screenStream = shallowRef(null)
  const isMuted = shallowRef(false)
  /** «Глухой режим»: выкл. микрофон + заглушить звук участников (и стрим). */
  const isDeafened = shallowRef(false)
  /** Сохранённое состояние микрофона до включения глухого режима. */
  const micMutedBeforeDeafen = shallowRef(false)
  const isScreenSharing = shallowRef(false)
  const localSpeaking = shallowRef(false)
  /** Последний момент, когда локальный VAD видел речь (для индикатора «тишина»). */
  const localLastVoiceAt = shallowRef(Date.now())
  const activeScreenSharer = shallowRef(null)
  const connected = shallowRef(false)
  const error = shallowRef(null)
  const hasJoined = shallowRef(false)

  const chatMessages = ref(loadChatHistory())
  const chatUnread = ref(0)

  // ─── Music state (from server via WS + REST on join) ────────────────────────
  const musicState = ref({ playing: false, current: null, queue: [] })

  async function fetchMusicState() {
    try {
      const res = await fetch('/api/music/state')
      if (res.ok) musicState.value = await res.json()
    } catch {}
  }

  // ─── Quiz states (server-driven, received via WebSocket broadcasts) ──────────
  const quizState      = ref(null)  // anime quiz  — QuizState or null
  const musicQuizState = ref(null)  // music quiz  — MusicQuizState or null

  async function fetchQuizStates() {
    try {
      const [qr, mr] = await Promise.allSettled([
        fetch('/api/quiz/state'),
        fetch('/api/musicquiz/state'),
      ])
      if (qr.status === 'fulfilled' && qr.value.ok) quizState.value = await qr.value.json()
      if (mr.status === 'fulfilled' && mr.value.ok) musicQuizState.value = await mr.value.json()
    } catch {}
  }

  // ─── Screen share settings (persisted) ──────────────────────────────────────
  const SCREEN_SETTINGS_KEY = 'zvonok_screen_settings_v1'
  const screenShareSettings = reactive({
    resolution: 'auto',  // 'auto' | '1920x1080' | '1280x720' | '854x480' | '640x360'
    fps: 30,             // 5 | 15 | 30 | 60
    detail: 'balanced',  // 'low' | 'balanced' | 'high' — авто-битрейт с запасом
    ...(() => {
      try {
        const raw = JSON.parse(localStorage.getItem(SCREEN_SETTINGS_KEY) ?? '{}')
        // migrate legacy bitrate (kbps) → detail preset
        if (raw && 'bitrate' in raw && !('detail' in raw)) {
          const br = raw.bitrate
          delete raw.bitrate
          raw.detail = br === 0 || br >= 15000 ? 'high' : br >= 4000 ? 'balanced' : 'low'
        }
        return raw
      } catch { return {} }
    })(),
  })
  watch(screenShareSettings, (s) => {
    localStorage.setItem(SCREEN_SETTINGS_KEY, JSON.stringify({ ...s }))
  }, { deep: true })

  let bitrateApplyTimer = null

  /** Авто-битрейт по разрешению, FPS и уровню детализации (с ~15% запасом). */
  function computeVideoBitrateKbps(s) {
    const { resolution, fps, detail } = s
    let w = 1920
    let h = 1080
    if (resolution && resolution !== 'auto') {
      const parts = resolution.split('x').map(Number)
      if (parts.length === 2 && parts.every((n) => Number.isFinite(n) && n > 0)) {
        ;[w, h] = parts
      }
    }
    const pixels = w * h
    const detailMul = detail === 'low' ? 0.58 : detail === 'high' ? 1.42 : 1
    // ~0.00008 kbps per pixel·frame at balanced 1080p30 → ~5 Mbit/s; capped for uplink safety
    let kbps = pixels * fps * 0.00008 * detailMul * 1.15
    kbps = Math.round(Math.min(40000, Math.max(600, kbps)))
    return kbps
  }

  async function applyBitrateToAllPeers(kbps) {
    for (const peerId of Object.keys(peerMeta)) {
      await applyBitrateToPc(peerMeta[peerId].pc, kbps)
    }
  }

  watch(
    screenShareSettings,
    () => {
      if (!isScreenSharing.value) return
      clearTimeout(bitrateApplyTimer)
      bitrateApplyTimer = setTimeout(() => {
        bitrateApplyTimer = null
        const kbps = computeVideoBitrateKbps(screenShareSettings)
        applyBitrateToAllPeers(kbps)
      }, 350)
    },
    { deep: true },
  )

  // Track whether the chat panel is open so we skip the unread counter
  let chatPanelOpen = false
  function setChatOpen(val) {
    chatPanelOpen = val
    if (val) chatUnread.value = 0
  }

  // Persist messages to localStorage on every change
  watch(chatMessages, (msgs) => saveChatHistory(msgs), { deep: false })

  let ws = null
  let audioCtx = null
  let analyser = null
  let vadTimer = null
  let prevSpeaking = false
  let reconnectTimer = null

  const participantsList = computed(() => Array.from(participants.values()))
  const screenSharersList = computed(() =>
    participantsList.value.filter((p) => p.hasScreenShare),
  )

  // ─── WebSocket Signaling ────────────────────────────────────────────────────

  function connectSignaling(name, avatar) {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    ws = new WebSocket(`${protocol}//${location.host}/ws`)

    ws.onopen = () => {
      connected.value = true
      ws.send(JSON.stringify({ type: 'join', payload: { name, avatar } }))
    }

    ws.onclose = () => {
      connected.value = false
      if (hasJoined.value) scheduleReconnect()
    }

    ws.onerror = (e) => {
      console.error('WS error:', e)
    }

    ws.onmessage = (e) => {
      try {
        handleSignalingMessage(JSON.parse(e.data))
      } catch (err) {
        console.error('WS message parse error:', err)
      }
    }
  }

  function scheduleReconnect() {
    if (reconnectTimer) return
    reconnectTimer = setTimeout(async () => {
      reconnectTimer = null
      if (!hasJoined.value) return

      // Cleanup stale peer connections
      for (const peerId of Object.keys(peerMeta)) {
        peerMeta[peerId].pc.close()
        delete peerMeta[peerId]
      }
      participants.forEach((p) => {
        if (p.audioEl) { p.audioEl.pause(); p.audioEl.srcObject = null }
        if (p.screenAudioEl) { p.screenAudioEl.pause(); p.screenAudioEl.srcObject = null }
      })
      participants.clear()

      connectSignaling(myInfo.value.name, myInfo.value.avatar)
    }, 2500)
  }

  function sendSignal(type, to, payload) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return
    ws.send(JSON.stringify({ type, to: to || undefined, payload }))
  }

  // ─── Signaling Message Handler ──────────────────────────────────────────────

  async function handleSignalingMessage(msg) {
    switch (msg.type) {
      case 'welcome': {
        myId.value = msg.payload.id
        hasJoined.value = true
        for (const p of msg.payload.participants || []) {
          addParticipant(p)
          await initPeerConnection(p.id)
          addLocalTracksToPeer(p.id)
        }
        // Re-broadcast our current state to peers that may have missed it
        if (isMuted.value) sendSignal('muted', null, { muted: true })
        if (myStatus.value) sendSignal('status', null, { status: myStatus.value })
        // Fetch current music queue state and quiz states from server
        fetchMusicState()
        fetchQuizStates()
        break
      }

      case 'peer-joined': {
        const p = msg.payload
        addParticipant(p)
        await initPeerConnection(p.id)
        addLocalTracksToPeer(p.id)
        break
      }

      case 'peer-left': {
        const id = msg.payload.id
        closePeerConnection(id)
        const p = participants.get(id)
        if (p?.audioEl) { p.audioEl.pause(); p.audioEl.srcObject = null }
        if (p?.screenAudioEl) { p.screenAudioEl.pause(); p.screenAudioEl.srcObject = null }
        participants.delete(id)
        if (activeScreenSharer.value === id) {
          const others = screenSharersList.value.filter((pp) => pp.id !== id)
          activeScreenSharer.value = others[0]?.id ?? null
        }
        break
      }

      case 'offer':  await handleOffer(msg.from, msg.payload); break
      case 'answer': await handleAnswer(msg.from, msg.payload); break
      case 'ice-candidate': await handleIceCandidate(msg.from, msg.payload); break

      case 'speaking': {
        const p = participants.get(msg.from)
        if (p) {
          p.speaking = msg.payload.speaking
          // Любое изменение VAD — активность по микрофону (начало или конец фразы).
          p.lastVoiceActivityAt = Date.now()
        }
        break
      }

      case 'muted': {
        const p = participants.get(msg.from)
        if (p) p.muted = msg.payload.muted
        break
      }

      case 'status': {
        const p = participants.get(msg.from)
        if (p) p.status = msg.payload.status
        break
      }

      case 'screen-share': {
        const p = participants.get(msg.from)
        if (p && !msg.payload.active) {
          p.hasScreenShare = false
          p.hasScreenAudio = false
          p.screenStream = null
          if (p.screenAudioEl) {
            p.screenAudioEl.pause()
            p.screenAudioEl.srcObject = null
          }
          if (activeScreenSharer.value === msg.from) {
            const others = screenSharersList.value.filter((pp) => pp.id !== msg.from)
            activeScreenSharer.value = others[0]?.id ?? null
          }
        }
        break
      }

      case 'music-state': {
        musicState.value = msg.payload
        break
      }

      case 'quiz-state': {
        quizState.value = msg.payload
        break
      }

      case 'musicquiz-state': {
        musicQuizState.value = msg.payload
        break
      }

      case 'chat': {
        const sender = participants.get(msg.from)
        chatMessages.value.push({
          id: `${msg.from}-${Date.now()}`,
          from: msg.from,
          isLocal: false,
          name: sender?.name ?? 'Участник',
          avatar: sender?.avatar ?? '',
          text: msg.payload.text,
          timestamp: msg.payload.timestamp ?? Date.now(),
        })
        // Only count unread when the chat panel is closed
        if (!chatPanelOpen) chatUnread.value++
        break
      }

      case 'game': {
        const { ns, event, data } = msg.payload ?? {}
        _gameHandlers.forEach((h) => h(msg.from, ns, event, data ?? {}))
        break
      }
    }
  }

  // ─── Participant Management ─────────────────────────────────────────────────

  function addParticipant(info) {
    if (!participants.has(info.id)) {
      participants.set(info.id, {
        id: info.id,
        name: info.name,
        avatar: info.avatar,
        speaking: false,
        muted: false,
        lastVoiceActivityAt: Date.now(),
        status: null,
        hasScreenShare: false,
        hasScreenAudio: false,
        audioStream: null,
        screenStream: null,
        volume: 1.0,
        audioEl: null,
        screenAudioEl: null,
      })
    }
  }

  function updateParticipant(id, updates) {
    const p = participants.get(id)
    if (p) Object.assign(p, updates)
  }

  // ─── WebRTC ─────────────────────────────────────────────────────────────────

  async function initPeerConnection(peerId) {
    if (peerMeta[peerId]) return peerMeta[peerId].pc

    const iceConfig = await fetchIceServers()
    if (peerMeta[peerId]) return peerMeta[peerId].pc

    const pc = new RTCPeerConnection(iceConfig)
    const isPolite = myId.value < peerId
    const meta = {
      pc,
      makingOffer: false,
      ignoreOffer: false,
      isPolite,
      negotiationVersion: 0,
      lastNegotiatedVersion: -1,
    }
    peerMeta[peerId] = meta

    pc.onnegotiationneeded = async () => {
      if (meta.negotiationVersion <= meta.lastNegotiatedVersion) return
      try {
        meta.makingOffer = true
        await pc.setLocalDescription()
        meta.lastNegotiatedVersion = meta.negotiationVersion
        sendSignal('offer', peerId, { sdp: pc.localDescription })
      } catch (e) {
        console.error(`[${peerId}] negotiation error:`, e)
      } finally {
        meta.makingOffer = false
      }
    }

    pc.onicecandidate = ({ candidate }) => {
      if (candidate) sendSignal('ice-candidate', peerId, { candidate })
    }

    pc.ontrack = ({ track, streams }) => {
      const stream = streams[0] ?? new MediaStream([track])
      if (track.kind === 'video') {
        updateParticipant(peerId, { hasScreenShare: true, screenStream: markRaw(stream) })
        if (!activeScreenSharer.value) activeScreenSharer.value = peerId
        recoverMicIfScreenAudioReplacedMicInAudioEl(peerId, stream)
        track.onended = () => {
          const p = participants.get(peerId)
          if (p?.screenAudioEl) { p.screenAudioEl.pause(); p.screenAudioEl.srcObject = null }
          updateParticipant(peerId, { hasScreenShare: false, hasScreenAudio: false, screenStream: null })
          if (activeScreenSharer.value === peerId) {
            const others = screenSharersList.value.filter((p) => p.id !== peerId)
            activeScreenSharer.value = others[0]?.id ?? null
          }
        }
      } else if (track.kind === 'audio') {
        const p = participants.get(peerId)
        // Звук вкладки/системы — тот же MediaStream, что и видео экрана: отдельный <audio>.
        if (p?.screenStream && p.screenStream.id === stream.id) {
          setupScreenAudioPlayback(peerId, track)
        } else if (!p?.screenStream) {
          // Видео экрана ещё нет: второй аудио-поток не должен вытеснять микрофон из audioEl.
          const curSid = p?.audioEl?.srcObject?.id
          if (curSid && curSid !== stream.id) {
            if (likelyDisplayCaptureAudio(track)) {
              setupScreenAudioPlayback(peerId, track)
            } else {
              const curTrack = p.audioEl?.srcObject?.getAudioTracks?.()[0]
              if (curTrack && likelyDisplayCaptureAudio(curTrack)) {
                setupScreenAudioPlayback(peerId, curTrack)
                p.audioEl.pause()
                p.audioEl.srcObject = null
              }
              updateParticipant(peerId, { audioStream: markRaw(stream) })
              setupAudioPlayback(peerId, stream)
            }
          } else if (likelyDisplayCaptureAudio(track)) {
            setupScreenAudioPlayback(peerId, track)
          } else {
            updateParticipant(peerId, { audioStream: markRaw(stream) })
            setupAudioPlayback(peerId, stream)
          }
        } else {
          updateParticipant(peerId, { audioStream: markRaw(stream) })
          setupAudioPlayback(peerId, stream)
        }
      }
    }

    pc.onconnectionstatechange = () => {
      if (pc.connectionState === 'failed') {
        meta.negotiationVersion++
        pc.restartIce()
      }
    }

    return pc
  }

  function addLocalTracksToPeer(peerId) {
    const meta = peerMeta[peerId]
    if (!meta) return
    const { pc } = meta

    if (localStream.value) {
      localStream.value.getTracks().forEach((track) => {
        if (!pc.getSenders().find((s) => s.track === track)) {
          try {
            pc.addTrack(track, localStream.value)
            meta.negotiationVersion++
          } catch (e) {
            console.error(`[addTrack] FAILED peer=${peerId}:`, e)
          }
        }
      })
    }

    if (screenStream.value) {
      screenStream.value.getTracks().forEach((track) => {
        if (!pc.getSenders().find((s) => s.track === track)) {
          try {
            pc.addTrack(track, screenStream.value)
            meta.negotiationVersion++
          } catch (e) {}
        }
      })
    }
  }

  function closePeerConnection(peerId) {
    const meta = peerMeta[peerId]
    if (meta) { meta.pc.close(); delete peerMeta[peerId] }
  }

  // ─── Perfect Negotiation ────────────────────────────────────────────────────

  async function handleOffer(fromId, { sdp }) {
    let meta = peerMeta[fromId]
    if (!meta) {
      await initPeerConnection(fromId)
      meta = peerMeta[fromId]
      addLocalTracksToPeer(fromId)
    }

    const { pc, isPolite } = meta
    const offerCollision =
      sdp.type === 'offer' && (meta.makingOffer || pc.signalingState !== 'stable')

    meta.ignoreOffer = !isPolite && offerCollision
    if (meta.ignoreOffer) return

    try {
      await pc.setRemoteDescription(sdp)
      if (sdp.type === 'offer') {
        await pc.setLocalDescription()
        meta.lastNegotiatedVersion = meta.negotiationVersion
        sendSignal('answer', fromId, { sdp: pc.localDescription })
      }
    } catch (e) {
      console.error(`[${fromId}] handleOffer error:`, e)
    }
  }

  async function handleAnswer(fromId, { sdp }) {
    const meta = peerMeta[fromId]
    if (!meta) return
    if (meta.pc.signalingState !== 'have-local-offer') return
    try {
      await meta.pc.setRemoteDescription(sdp)
    } catch (e) {
      console.warn(`[${fromId}] handleAnswer error:`, e)
    }
  }

  async function handleIceCandidate(fromId, { candidate }) {
    const meta = peerMeta[fromId]
    if (!meta || !candidate) return
    try {
      await meta.pc.addIceCandidate(candidate)
    } catch (e) {
      if (!meta.ignoreOffer) console.warn(`[${fromId}] ICE error:`, e)
    }
  }

  // ─── Audio Playback ─────────────────────────────────────────────────────────

  function setupAudioPlayback(peerId, stream) {
    const p = participants.get(peerId)
    if (!p) { setTimeout(() => setupAudioPlayback(peerId, stream), 100); return }

    if (p.audioEl) {
      p.audioEl.srcObject = stream
      p.audioEl.play().catch(() => {})
      return
    }

    const audio = new Audio()
    audio.srcObject = stream
    audio.volume = p.volume
    audio.autoplay = true
    audio.play().catch(() => {
      const unlock = () => audio.play().catch(() => {})
      document.addEventListener('click', unlock, { once: true })
      document.addEventListener('keydown', unlock, { once: true })
    })
    audio.muted = isDeafened.value
    p.audioEl = markRaw(audio)
  }

  function setupScreenAudioPlayback(peerId, track) {
    const p = participants.get(peerId)
    if (!p) {
      setTimeout(() => setupScreenAudioPlayback(peerId, track), 100)
      return
    }

    const stream = new MediaStream([track])
    p.hasScreenAudio = true
    track.onended = () => {
      const p2 = participants.get(peerId)
      if (!p2) return
      p2.hasScreenAudio = false
      if (p2.screenAudioEl) {
        p2.screenAudioEl.pause()
        p2.screenAudioEl.srcObject = null
      }
    }

    if (p.screenAudioEl) {
      p.screenAudioEl.srcObject = stream
      p.screenAudioEl.volume = p.volume ?? 1
      p.screenAudioEl.muted = isDeafened.value
      p.screenAudioEl.play().catch(() => {})
      return
    }

    const audio = new Audio()
    audio.srcObject = stream
    audio.volume = p.volume ?? 1
    audio.muted = isDeafened.value
    audio.autoplay = true
    audio.play().catch(() => {
      const unlock = () => audio.play().catch(() => {})
      document.addEventListener('click', unlock, { once: true })
      document.addEventListener('keydown', unlock, { once: true })
    })
    p.screenAudioEl = markRaw(audio)
  }

  /**
   * Если звук экрана попал в audioEl до прихода видео (один MediaStream), переносим его в screenAudioEl
   * и подключаем микрофон с другого receiver.
   */
  function recoverMicIfScreenAudioReplacedMicInAudioEl(peerId, screenStream) {
    const p = participants.get(peerId)
    const meta = peerMeta[peerId]
    if (!p || !meta?.pc) return

    const screenAudio = screenStream.getAudioTracks()[0]
    if (!screenAudio) return

    if (p.audioEl?.srcObject?.id !== screenStream.id) return

    setupScreenAudioPlayback(peerId, screenAudio)
    p.audioEl.pause()
    p.audioEl.srcObject = null

    for (const r of meta.pc.getReceivers()) {
      const t = r.track
      if (!t || t.kind !== 'audio' || t.id === screenAudio.id) continue
      const ms = new MediaStream([t])
      updateParticipant(peerId, { audioStream: markRaw(ms) })
      setupAudioPlayback(peerId, ms)
      break
    }
  }

  // ─── VAD ───────────────────────────────────────────────────────────────────

  function startVAD(stream) {
    try {
      audioCtx = new AudioContext()
      analyser = audioCtx.createAnalyser()
      analyser.fftSize = 512
      audioCtx.createMediaStreamSource(stream).connect(analyser)
      const data = new Uint8Array(analyser.frequencyBinCount)
      vadTimer = setInterval(() => {
        if (isMuted.value) {
          if (prevSpeaking) { prevSpeaking = false; localSpeaking.value = false; sendSignal('speaking', null, { speaking: false }) }
          return
        }
        analyser.getByteFrequencyData(data)
        const speaking = data.reduce((a, b) => a + b, 0) / data.length > 8
        if (speaking !== prevSpeaking) {
          prevSpeaking = speaking
          localSpeaking.value = speaking
          sendSignal('speaking', null, { speaking })
          localLastVoiceAt.value = Date.now()
        }
      }, 150)
    } catch (e) { console.warn('VAD setup error:', e) }
  }

  function stopVAD() {
    if (vadTimer) clearInterval(vadTimer)
    vadTimer = null
    if (audioCtx) { audioCtx.close(); audioCtx = null }
    analyser = null; prevSpeaking = false; localSpeaking.value = false
  }

  // ─── Public Actions ─────────────────────────────────────────────────────────

  async function join(name, avatar) {
    myInfo.value = { name, avatar }
    error.value = null
    await fetchIceServers()
    try {
      localStream.value = await navigator.mediaDevices.getUserMedia({
        audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true },
        video: false,
      })
      startVAD(localStream.value)
      localLastVoiceAt.value = Date.now()
    } catch (e) {
      console.warn('Microphone denied:', e)
      error.value = 'Нет доступа к микрофону. Вы можете слушать, но не говорить.'
    }
    connectSignaling(name, avatar)
  }

  function setMuted(nextMuted) {
    if (isDeafened.value && nextMuted === false) return
    if (isMuted.value === nextMuted) return
    isMuted.value = nextMuted
    if (localStream.value) {
      localStream.value.getAudioTracks().forEach((t) => { t.enabled = !isMuted.value })
    }
    if (isMuted.value) { localSpeaking.value = false; sendSignal('speaking', null, { speaking: false }) }
    sendSignal('muted', null, { muted: isMuted.value })
  }

  function toggleMute() {
    if (isDeafened.value) return
    setMuted(!isMuted.value)
  }

  function syncRemoteOutputMute() {
    const m = isDeafened.value
    participants.forEach((p) => {
      if (p.audioEl) p.audioEl.muted = m
      if (p.screenAudioEl) p.screenAudioEl.muted = m
    })
  }

  function setDeafened(on) {
    if (on === isDeafened.value) return
    if (on) {
      micMutedBeforeDeafen.value = isMuted.value
      isDeafened.value = true
      setMuted(true)
      syncRemoteOutputMute()
    } else {
      isDeafened.value = false
      syncRemoteOutputMute()
      setMuted(micMutedBeforeDeafen.value)
    }
  }

  function toggleDeafen() {
    setDeafened(!isDeafened.value)
  }

  function setStatus(status) {
    myStatus.value = status
    sendSignal('status', null, { status })
  }

  // ─── Game Event System ───────────────────────────────────────────────────────
  // Game events are broadcast to all peers via the signaling channel.
  // They are NOT displayed in chat — the 'game' type is filtered out above.

  const _gameHandlers = []

  function sendGameEvent(ns, event, data = {}) {
    sendSignal('game', null, { ns, event, data })
  }

  function onGameEvent(handler) {
    _gameHandlers.push(handler)
    return () => {
      const i = _gameHandlers.indexOf(handler)
      if (i !== -1) _gameHandlers.splice(i, 1)
    }
  }

  // ─── Chat ────────────────────────────────────────────────────────────────────

  function sendChatMessage(text) {
    if (!text.trim()) return
    const timestamp = Date.now()
    sendSignal('chat', null, { text: text.trim(), timestamp })
    chatMessages.value.push({
      id: `local-${timestamp}`,
      from: myId.value,
      isLocal: true,
      name: myInfo.value?.name ?? 'Я',
      avatar: myInfo.value?.avatar ?? '',
      text: text.trim(),
      timestamp,
    })
  }

  function clearChatUnread() {
    chatUnread.value = 0
  }

  // Apply max bitrate to all video senders of a peer connection
  async function applyBitrateToPc(pc, kbps) {
    const videoSenders = pc.getSenders().filter((s) => s.track?.kind === 'video')
    for (const sender of videoSenders) {
      try {
        const params = sender.getParameters()
        if (!params.encodings?.length) params.encodings = [{}]
        params.encodings.forEach((enc) => {
          if (kbps > 0) enc.maxBitrate = kbps * 1000
          else delete enc.maxBitrate
        })
        await sender.setParameters(params)
      } catch (e) {
        console.warn('[bitrate] setParameters failed:', e)
      }
    }
  }

  async function startScreenShare() {
    try {
      const { resolution, fps } = screenShareSettings

      // Build video constraints
      const videoConstraints = {
        cursor: 'always',
        frameRate: { ideal: fps, max: fps },
      }
      if (resolution !== 'auto') {
        const [w, h] = resolution.split('x').map(Number)
        videoConstraints.width  = { ideal: w, max: w }
        videoConstraints.height = { ideal: h, max: h }
      }

      const stream = await navigator.mediaDevices.getDisplayMedia({
        video: videoConstraints,
        audio: true,
      })
      screenStream.value = stream
      isScreenSharing.value = true

      for (const peerId of Object.keys(peerMeta)) {
        const meta = peerMeta[peerId]
        stream.getTracks().forEach((track) => {
          if (!meta.pc.getSenders().find((s) => s.track === track)) {
            meta.pc.addTrack(track, stream)
            meta.negotiationVersion++
          }
        })
        const kbps = computeVideoBitrateKbps(screenShareSettings)
        setTimeout(() => applyBitrateToPc(meta.pc, kbps), 1500)
      }

      sendSignal('screen-share', null, { active: true })
      stream.getVideoTracks()[0].onended = () => stopScreenShare()
    } catch (e) {
      if (e.name !== 'NotAllowedError') { console.error('Screen share error:', e); error.value = 'Не удалось запустить трансляцию экрана' }
    }
  }

  async function stopScreenShare() {
    const stream = screenStream.value
    if (!stream) return
    const screenTrackIds = new Set(stream.getTracks().map((t) => t.id))
    for (const peerId of Object.keys(peerMeta)) {
      const meta = peerMeta[peerId]
      meta.pc.getSenders().forEach((sender) => {
        if (sender.track && screenTrackIds.has(sender.track.id)) {
          meta.pc.removeTrack(sender)
          meta.negotiationVersion++
        }
      })
    }
    stream.getTracks().forEach((t) => t.stop())
    screenStream.value = null
    isScreenSharing.value = false
    sendSignal('screen-share', null, { active: false })
  }

  function setParticipantVolume(peerId, volume) {
    const p = participants.get(peerId)
    if (!p) return
    p.volume = volume
    if (p.audioEl) p.audioEl.volume = volume
    if (p.screenAudioEl) p.screenAudioEl.volume = volume
  }

  function leave() {
    if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null }
    if (bitrateApplyTimer) { clearTimeout(bitrateApplyTimer); bitrateApplyTimer = null }
    stopVAD()
    if (localStream.value) { localStream.value.getTracks().forEach((t) => t.stop()); localStream.value = null }
    const stream = screenStream.value
    if (stream) { stream.getTracks().forEach((t) => t.stop()); screenStream.value = null }
    for (const peerId of Object.keys(peerMeta)) { peerMeta[peerId].pc.close(); delete peerMeta[peerId] }
    participants.forEach((p) => {
      if (p.audioEl) { p.audioEl.pause(); p.audioEl.srcObject = null }
      if (p.screenAudioEl) { p.screenAudioEl.pause(); p.screenAudioEl.srcObject = null }
    })
    participants.clear()
    if (ws) { ws.close(); ws = null }
    myId.value = null; myInfo.value = null; hasJoined.value = false; connected.value = false
    isMuted.value = false
    isDeafened.value = false
    isScreenSharing.value = false
    activeScreenSharer.value = null
    error.value = null; myStatus.value = null; chatUnread.value = 0
    // Don't clear chatMessages on leave — keep history in localStorage
  }

  return {
    myId, myInfo, myStatus,
    participants, participantsList, screenSharersList,
    localStream, screenStream,
    isMuted, isDeafened, isScreenSharing, localSpeaking, localLastVoiceAt, activeScreenSharer,
    connected, error, hasJoined,
    chatMessages, chatUnread,
    screenShareSettings,
    musicState,
    join, leave, toggleMute, setMuted, toggleDeafen,
    setStatus, sendChatMessage, clearChatUnread, setChatOpen,
    startScreenShare, stopScreenShare, setParticipantVolume,
    sendGameEvent, onGameEvent,
    quizState, musicQuizState,
  }
}
