class NotificationManager {
    constructor() {
        this.notifications = [];
        this.toasts = [];
        this.maxNotifications = 20;
    }

    init() {
        this.createNotificationPanel();
        this.createToastContainer();
        this.createConnectionStatus();

        // Setup WebSocket listeners
        wsClient.on('order_created', (data) => this.onOrderCreated(data));
        wsClient.on('order_status', (data) => this.onOrderStatusChanged(data));
        wsClient.on('notification', (data) => this.onNotification(data));
        wsClient.on('connected', () => this.onConnected());
        wsClient.on('disconnected', () => this.onDisconnected());
    }

    createNotificationPanel() {
        const panel = document.createElement('div');
        panel.id = 'notificationPanel';
        panel.className = 'notification-panel';
        panel.innerHTML = `
            <div class="notification-header">
                <h3>Powiadomienia</h3>
                <button class="clear-notifications" onclick="notificationManager.clearAll()">Wyczyść</button>
            </div>
            <div class="notification-list" id="notificationList">
                <div class="notification-empty">Brak powiadomień</div>
            </div>
        `;
        document.body.appendChild(panel);

        // Close on click outside
        document.addEventListener('click', (e) => {
            if (!e.target.closest('.notification-icon') && !e.target.closest('#notificationPanel')) {
                this.closePanel();
            }
        });
    }

    createToastContainer() {
        const container = document.createElement('div');
        container.id = 'toastContainer';
        container.className = 'toast-container';
        document.body.appendChild(container);
    }

    createConnectionStatus() {
        const status = document.createElement('div');
        status.id = 'connectionStatus';
        status.className = 'connection-status';
        status.innerHTML = `
            <div class="status-dot"></div>
            <span>Łączę się...</span>
        `;
        document.body.appendChild(status);
    }

    addNotification(notif) {
        this.notifications.unshift(notif);

        if (this.notifications.length > this.maxNotifications) {
            this.notifications = this.notifications.slice(0, this.maxNotifications);
        }

        this.updateNotificationList();
        this.updateBadge();
    }

    updateNotificationList() {
        const list = document.getElementById('notificationList');
        const unreadCount = this.notifications.filter(n => !n.read).length;

        if (this.notifications.length === 0) {
            list.innerHTML = '<div class="notification-empty">Brak powiadomień</div>';
            return;
        }

        list.innerHTML = this.notifications.map(notif => `
            <div class="notification-item ${notif.read ? '' : 'unread'} ${notif.type}">
                <div class="notification-icon-circle">
                    ${notif.type === 'order_created' ? '✓' : notif.type === 'order_status' ? '📦' : '🔔'}
                </div>
                <div class="notification-content">
                    <div class="notification-title">${notif.title}</div>
                    <div class="notification-message">${notif.message}</div>
                    <div class="notification-time">${this.getTimeAgo(notif.created_at)}</div>
                </div>
            </div>
        `).join('');
    }

    updateBadge() {
        const unreadCount = this.notifications.filter(n => !n.read).length;
        let badge = document.querySelector('.notification-badge');

        if (unreadCount > 0) {
            if (!badge) {
                badge = document.createElement('div');
                badge.className = 'notification-badge';
                const icon = document.querySelector('.notification-icon');
                if (icon) icon.appendChild(badge);
            }
            badge.textContent = unreadCount > 99 ? '99+' : unreadCount;
        } else if (badge) {
            badge.remove();
        }
    }

    togglePanel() {
        const panel = document.getElementById('notificationPanel');
        panel.classList.toggle('active');

        if (panel.classList.contains('active')) {
            // Mark all as read
            this.notifications.forEach(n => n.read = true);
            this.updateNotificationList();
            this.updateBadge();
        }
    }

    closePanel() {
        const panel = document.getElementById('notificationPanel');
        panel.classList.remove('active');
    }

    clearAll() {
        this.notifications = [];
        this.updateNotificationList();
        this.updateBadge();
    }

    showToast(title, message, type = 'info') {
        const container = document.getElementById('toastContainer');
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;

        const icons = {
            success: '✓',
            error: '✕',
            info: 'ℹ',
        };

        toast.innerHTML = `
            <div class="toast-icon">${icons[type] || '🔔'}</div>
            <div class="toast-content">
                <div class="toast-title">${title}</div>
                <div class="toast-message">${message}</div>
            </div>
            <button class="toast-close" onclick="this.parentElement.remove()">✕</button>
        `;

        container.appendChild(toast);

        // Auto remove after 5 seconds
        setTimeout(() => {
            toast.remove();
        }, 5000);
    }

    onOrderCreated(data) {
        const notif = {
            type: 'order_created',
            title: 'Zamówienie złożone',
            message: `Zamówienie #${data.id} zostało złożone (${data.total_price.toFixed(2)} zł)`,
            order_id: data.id,
            read: false,
            created_at: new Date().toISOString(),
        };

        this.addNotification(notif);
        this.showToast('Zamówienie złożone', `Zamówienie #${data.id} zostało przyjęte`, 'success');
    }

    onOrderStatusChanged(data) {
        const statusLabels = {
            'pending': 'Oczekujące',
            'confirmed': 'Potwierdzone',
            'shipped': 'Wysłane',
            'delivered': 'Dostarczone',
            'cancelled': 'Anulowane',
        };

        const notif = {
            type: 'order_status',
            title: 'Status zamówienia',
            message: `Zamówienie #${data.id} - Status: ${statusLabels[data.status] || data.status}`,
            order_id: data.id,
            read: false,
            created_at: new Date().toISOString(),
        };

        this.addNotification(notif);
        this.showToast('Aktualizacja zamówienia', notif.message, 'info');
    }

    onNotification(data) {
        data.read = false;
        data.created_at = data.created_at || new Date().toISOString();
        this.addNotification(data);
    }

    onConnected() {
        const status = document.getElementById('connectionStatus');
        status.innerHTML = `
            <div class="status-dot connected"></div>
            <span>Połączono</span>
        `;
        setTimeout(() => {
            status.classList.remove('visible');
        }, 2000);
    }

    onDisconnected() {
        const status = document.getElementById('connectionStatus');
        status.classList.add('visible');
        status.innerHTML = `
            <div class="status-dot"></div>
            <span>Łączę się...</span>
        `;
    }

    getTimeAgo(dateString) {
        const date = new Date(dateString);
        const now = new Date();
        const seconds = Math.floor((now - date) / 1000);

        if (seconds < 60) return 'przed chwilą';
        if (seconds < 3600) return `${Math.floor(seconds / 60)}m temu`;
        if (seconds < 86400) return `${Math.floor(seconds / 3600)}h temu`;
        if (seconds < 604800) return `${Math.floor(seconds / 86400)}d temu`;

        return date.toLocaleDateString('pl-PL');
    }
}

const notificationManager = new NotificationManager();
