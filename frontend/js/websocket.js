const WS_URL = 'ws://localhost:8080/ws';

class WebSocketClient {
    constructor() {
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 3000;
        this.listeners = {};
        this.connected = false;
    }

    connect(token) {
        return new Promise((resolve, reject) => {
            try {
                const wsURL = `${WS_URL}?token=${encodeURIComponent(token)}`;
                this.ws = new WebSocket(wsURL);

                this.ws.onopen = () => {
                    this.connected = true;
                    this.reconnectAttempts = 0;
                    console.log('✓ WebSocket connected');
                    this.emit('connected');
                    resolve();
                };

                this.ws.onmessage = (event) => {
                    try {
                        const message = JSON.parse(event.data);
                        this.handleMessage(message);
                    } catch (error) {
                        console.error('Failed to parse WebSocket message:', error);
                    }
                };

                this.ws.onerror = (error) => {
                    console.error('WebSocket error:', error);
                    this.connected = false;
                    reject(error);
                };

                this.ws.onclose = () => {
                    this.connected = false;
                    console.log('✗ WebSocket disconnected');
                    this.emit('disconnected');
                    this.attemptReconnect(token);
                };

                // Set timeout for connection
                setTimeout(() => {
                    if (!this.connected) {
                        reject(new Error('Connection timeout'));
                    }
                }, 5000);
            } catch (error) {
                reject(error);
            }
        });
    }

    handleMessage(message) {
        switch (message.type) {
            case 'pong':
                // Respond to pong
                break;
            case 'order_created':
                this.emit('order_created', message.data);
                break;
            case 'order_status':
                this.emit('order_status', message.data);
                break;
            case 'notification':
                this.emit('notification', message.data);
                break;
            default:
                this.emit(message.type, message.data);
        }
    }

    send(type, data = {}) {
        if (!this.connected) {
            console.warn('WebSocket not connected');
            return;
        }

        const message = {
            type,
            timestamp: new Date().toISOString(),
            data,
        };

        this.ws.send(JSON.stringify(message));
    }

    on(event, callback) {
        if (!this.listeners[event]) {
            this.listeners[event] = [];
        }
        this.listeners[event].push(callback);
    }

    off(event, callback) {
        if (this.listeners[event]) {
            this.listeners[event] = this.listeners[event].filter(cb => cb !== callback);
        }
    }

    emit(event, data) {
        if (this.listeners[event]) {
            this.listeners[event].forEach(callback => {
                try {
                    callback(data);
                } catch (error) {
                    console.error(`Error in listener for ${event}:`, error);
                }
            });
        }
    }

    ping() {
        this.send('ping');
    }

    attemptReconnect(token) {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            console.log(`Attempting reconnect ${this.reconnectAttempts}/${this.maxReconnectAttempts}...`);
            setTimeout(() => {
                this.connect(token).catch(err => {
                    console.error('Reconnect failed:', err);
                });
            }, this.reconnectDelay);
        }
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
        }
    }
}

const wsClient = new WebSocketClient();
