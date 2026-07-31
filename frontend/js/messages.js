let currentUser = null;
let conversations = [];
let activeUserID = null;
let activeUserName = null;

function requireLogin() {
  const user = getCurrentUser();
  if (!user) {
    window.location.href = 'login.html?redirect=messages.html';
    return null;
  }
  return user;
}

function renderConversations() {
  const container = document.getElementById('conversationsList');
  if (conversations.length === 0) {
    container.innerHTML = '<p class="messages-empty" style="padding:16px;">Brak wiadomości.</p>';
    return;
  }
  container.innerHTML = conversations
    .map(
      (c) => `
    <a class="conversation-row ${c.user_id === activeUserID ? 'active' : ''}" href="#" data-id="${c.user_id}" data-name="${escapeHtml(c.username)}">
      <p class="conversation-row-name">
        <span>${escapeHtml(c.username)}</span>
        ${c.unread_count > 0 ? `<span class="conversation-unread-badge">${c.unread_count}</span>` : ''}
      </p>
      <p class="conversation-row-preview">${escapeHtml(c.last_body)}</p>
    </a>
  `
    )
    .join('');

  container.querySelectorAll('.conversation-row').forEach((row) => {
    row.addEventListener('click', (e) => {
      e.preventDefault();
      openThread(row.dataset.id, row.dataset.name);
    });
  });
}

function renderThread(messages) {
  const body = document.getElementById('threadBody');
  if (messages.length === 0) {
    body.innerHTML = '<p class="messages-empty">Brak wiadomości. Napisz pierwszą!</p>';
    return;
  }
  body.innerHTML = messages
    .map((m) => {
      const mine = m.sender_id === currentUser.id;
      const time = new Date(m.sent_at).toLocaleString('pl-PL', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' });
      return `
        <div class="message-bubble ${mine ? 'mine' : 'theirs'}">
          <p>${escapeHtml(m.body)}</p>
          <span class="message-bubble-time">${time}</span>
        </div>
      `;
    })
    .join('');
  body.scrollTop = body.scrollHeight;
}

async function loadConversations() {
  try {
    const res = await authFetch('/messages/conversations');
    if (!res.ok) return;
    conversations = await res.json();
    renderConversations();
  } catch (err) {
    // lista rozmów odświeży się przy kolejnej próbie
  }
}

async function openThread(userId, name) {
  activeUserID = userId;
  activeUserName = name;
  renderConversations();

  const header = document.getElementById('threadHeader');
  header.textContent = name;
  header.style.display = 'block';
  document.getElementById('messageForm').style.display = 'flex';
  document.getElementById('threadBody').innerHTML = '<p class="messages-empty">Ładowanie...</p>';

  try {
    const res = await authFetch(`/messages/with/${userId}`);
    if (!res.ok) {
      document.getElementById('threadBody').innerHTML = '<p class="messages-empty">Nie udało się pobrać wiadomości.</p>';
      return;
    }
    renderThread(await res.json());
    loadConversations();
    if (typeof refreshUnreadBadge === 'function') refreshUnreadBadge();
  } catch (err) {
    document.getElementById('threadBody').innerHTML = '<p class="messages-empty">Błąd ładowania wiadomości.</p>';
  }
}

function bindEvents() {
  document.getElementById('messageForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!activeUserID) return;
    const input = document.getElementById('messageInput');
    const body = input.value.trim();
    if (!body) return;
    const submitBtn = e.target.querySelector('button');
    submitBtn.disabled = true;
    try {
      const res = await authFetch('/messages', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ receiver_id: activeUserID, body }),
      });
      if (res.ok) {
        input.value = '';
        await openThread(activeUserID, activeUserName);
      }
    } finally {
      submitBtn.disabled = false;
    }
  });
}

// Fired by ws.js when the server pushes a new message over the socket.
function onWSChatMessage(message) {
  if (activeUserID && (message.sender_id === activeUserID || message.receiver_id === activeUserID)) {
    openThread(activeUserID, activeUserName);
  } else {
    loadConversations();
  }
}

async function init() {
  currentUser = requireLogin();
  if (!currentUser) return;
  updateHeaderForAuth();
  bindEvents();
  await loadConversations();

  const params = new URLSearchParams(window.location.search);
  const withId = params.get('with');
  if (withId) {
    const name = params.get('name') || (conversations.find((c) => c.user_id === withId) || {}).username || 'Rozmowa';
    openThread(withId, name);
  }
}

init();
