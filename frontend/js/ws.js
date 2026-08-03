// WebSocket Connection Management
// Establishes and maintains real-time communication for live notifications
// Automatically reconnects with exponential backoff on disconnection

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

  // Reset backoff delay on successful connection
  ws.onopen = () => {
    wsReconnectDelay = 1000;
  };

  // Handle incoming WebSocket messages
  ws.onmessage = (event) => {
    let msg;
    try {
      msg = JSON.parse(event.data);
    } catch (err) {
      return;
    }
    handleWSEvent(msg);
  };

  // Reconnect on close with exponential backoff (max 30 seconds)
  ws.onclose = () => {
    ws = null;
    clearTimeout(wsReconnectTimer);
    wsReconnectTimer = setTimeout(connectWS, wsReconnectDelay);
    wsReconnectDelay = Math.min(wsReconnectDelay * 2, 30000);
  };

  // Close the connection on error
  ws.onerror = () => {
    ws.close();
  };
}

// Cleanly closes the WebSocket connection and cancels any pending reconnection attempts
function disconnectWS() {
  clearTimeout(wsReconnectTimer);
  if (ws) {
    ws.onclose = null; // Prevent automatic reconnection
    ws.close();
    ws = null;
  }
}

// Routes incoming WebSocket messages to appropriate handlers
function handleWSEvent(msg) {
  // Update unread message badge when count changes
  if (msg.type === 'unread_count') {
    if (typeof setUnreadBadge === 'function') setUnreadBadge(msg.count);
  }
  // Notify messages page of incoming chat message
  else if (msg.type === 'message') {
    if (typeof onWSChatMessage === 'function') onWSChatMessage(msg.data);
  }
}
