let ws = null;
let wsReconnectTimer = null;
let wsReconnectDelay = 1000;

// Opens a live push channel for the current session (new messages, unread
// count changes). Falls back gracefully — reconnects with backoff, and the
// existing periodic refreshUnreadBadge() poll in nav.js still runs as a
// safety net if the socket is down for a while.
function connectWS() {
  const token = getAccessToken();
  if (!token) return;
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return;

  const wsUrl = `${API_URL.replace(/^http/, 'ws')}/ws?token=${encodeURIComponent(token)}`;
  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    wsReconnectDelay = 1000;
  };

  ws.onmessage = (event) => {
    let msg;
    try {
      msg = JSON.parse(event.data);
    } catch (err) {
      return;
    }
    handleWSEvent(msg);
  };

  ws.onclose = () => {
    ws = null;
    clearTimeout(wsReconnectTimer);
    wsReconnectTimer = setTimeout(connectWS, wsReconnectDelay);
    wsReconnectDelay = Math.min(wsReconnectDelay * 2, 30000);
  };

  ws.onerror = () => {
    ws.close();
  };
}

function disconnectWS() {
  clearTimeout(wsReconnectTimer);
  if (ws) {
    ws.onclose = null;
    ws.close();
    ws = null;
  }
}

function handleWSEvent(msg) {
  if (msg.type === 'unread_count') {
    if (typeof setUnreadBadge === 'function') setUnreadBadge(msg.count);
  } else if (msg.type === 'message') {
    if (typeof onWSChatMessage === 'function') onWSChatMessage(msg.data);
  }
}
