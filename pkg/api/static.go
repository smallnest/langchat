package api

import (
	"log"
	"net/http"
)

// StaticHandler serves static files and the main application
type StaticHandler struct {
	authAPI *AuthAPI
}

// NewStaticHandler creates a new static handler
func NewStaticHandler(authAPI *AuthAPI) *StaticHandler {
	return &StaticHandler{
		authAPI: authAPI,
	}
}

// ServeMainApp serves the main application HTML
func (s *StaticHandler) ServeMainApp(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(mainAppHTML)); err != nil {
		log.Printf("Warning: Failed to write main app HTML: %v", err)
	}
}

// mainAppHTML is the main application HTML with authentication
const mainAppHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Chat Agent - AI Assistant</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 0;
            background: #f5f5f5;
        }
        .header {
            background: white;
            padding: 1rem 2rem;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .header h1 {
            margin: 0;
            color: #333;
        }
        .user-info {
            display: flex;
            align-items: center;
            gap: 1rem;
        }
        .user-name {
            font-weight: 500;
            color: #666;
        }
        .logout-btn {
            background: #e74c3c;
            color: white;
            border: none;
            padding: 0.5rem 1rem;
            border-radius: 5px;
            cursor: pointer;
            text-decoration: none;
        }
        .logout-btn:hover {
            background: #c0392b;
        }
        .container {
            max-width: 1200px;
            margin: 2rem auto;
            padding: 0 2rem;
        }
        .chat-container {
            background: white;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            height: 70vh;
            display: flex;
            flex-direction: column;
        }
        .chat-messages {
            flex: 1;
            padding: 1rem;
            overflow-y: auto;
        }
        .chat-input {
            padding: 1rem;
            border-top: 1px solid #eee;
            display: flex;
            gap: 1rem;
        }
        .chat-input input {
            flex: 1;
            padding: 0.75rem;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-size: 1rem;
        }
        .chat-input button {
            background: #667eea;
            color: white;
            border: none;
            padding: 0.75rem 1.5rem;
            border-radius: 5px;
            cursor: pointer;
        }
        .chat-input button:hover {
            background: #5a6fd8;
        }
        .message {
            margin-bottom: 1rem;
            padding: 0.75rem;
            border-radius: 5px;
        }
        .message.user {
            background: #667eea;
            color: white;
            margin-left: 20%;
        }
        .message.assistant {
            background: #f8f9fa;
            color: #333;
            margin-right: 20%;
        }
        .status-bar {
            background: white;
            padding: 1rem;
            border-radius: 5px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 1rem;
        }
        .status-item {
            display: inline-block;
            margin-right: 2rem;
            font-size: 0.9rem;
            color: #666;
        }
        .status-item strong {
            color: #333;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1><img src="/static/images/logo.png" alt="Chat Agent" style="height: 40px; vertical-align: middle; margin-right: 10px;">Chat Agent</h1>
        <div class="user-info">
            <span class="user-name" id="user-name">Loading...</span>
            <button class="logout-btn" onclick="logout()">Logout</button>
        </div>
    </div>

    <div class="container">
        <div class="status-bar">
            <div class="status-item">
                <strong>Environment:</strong> <span id="environment">Development</span>
            </div>
            <div class="status-item">
                <strong>Model:</strong> <span id="model">GPT-4</span>
            </div>
            <div class="status-item">
                <strong>Agent Status:</strong> <span id="agent-status">Ready</span>
            </div>
        </div>

        <div class="chat-container">
            <div class="chat-messages" id="chat-messages">
                <div class="message assistant">
                    <strong>Assistant:</strong> Hello! I'm your AI assistant. How can I help you today?
                </div>
            </div>
            <div class="chat-input">
                <input type="text" id="message-input" placeholder="Type your message..." />
                <button onclick="sendMessage()">Send</button>
            </div>
        </div>
    </div>

    <script>
        // Check authentication on page load
        function checkAuth() {
            const token = localStorage.getItem('access_token');
            const user = localStorage.getItem('user');

            if (!token || !user) {
                window.location.href = '/login';
                return;
            }

            try {
                const userData = JSON.parse(user);
                document.getElementById('user-name').textContent = userData.username || 'User';
            } catch (error) {
                window.location.href = '/login';
            }
        }

        // Load app configuration
        async function loadConfig() {
            try {
                const response = await fetch('/api/config');
                const config = await response.json();

                document.getElementById('environment').textContent = config.environment || 'Unknown';
                document.getElementById('model').textContent = config.llmModel || 'Unknown';
            } catch (error) {
                console.error('Failed to load config:', error);
            }
        }

        // Check agent status
        async function checkAgentStatus() {
            try {
                const response = await fetch('/health');
                const status = await response.json();

                document.getElementById('agent-status').textContent =
                    status.status === 'healthy' ? 'Ready' : 'Error';
            } catch (error) {
                document.getElementById('agent-status').textContent = 'Error';
            }
        }

        // Send message
        async function sendMessage() {
            const input = document.getElementById('message-input');
            const message = input.value.trim();

            if (!message) return;

            const token = localStorage.getItem('access_token');
            if (!token) {
                window.location.href = '/login';
                return;
            }

            // Add user message to chat
            addMessage('user', message);
            input.value = '';

            try {
                const response = await fetch('/api/chat', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + token
                    },
                    body: JSON.stringify({ message: message })
                });

                if (response.status === 401) {
                    window.location.href = '/login';
                    return;
                }

                const data = await response.json();

                if (response.ok) {
                    addMessage('assistant', data.response || data.message || 'Sorry, I could not process your request.');
                } else {
                    addMessage('assistant', 'Error: ' + (data.error || 'Unknown error occurred.'));
                }
            } catch (error) {
                addMessage('assistant', 'Network error. Please try again.');
            }
        }

        // Add message to chat
        function addMessage(type, content) {
            const messagesDiv = document.getElementById('chat-messages');
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message ' + type;
            messageDiv.innerHTML = '<strong>' + (type === 'user' ? 'You' : 'Assistant') + ':</strong> ' + content;
            messagesDiv.appendChild(messageDiv);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }

        // Logout function
        async function logout() {
            const refreshToken = localStorage.getItem('refresh_token');

            if (refreshToken) {
                try {
                    await fetch('/api/auth/logout', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({ refresh_token: refreshToken })
                    });
                } catch (error) {
                    console.error('Logout error:', error);
                }
            }

            // Clear localStorage
            localStorage.removeItem('access_token');
            localStorage.removeItem('refresh_token');
            localStorage.removeItem('user');

            // Redirect to login
            window.location.href = '/login';
        }

        // Handle Enter key in message input
        document.getElementById('message-input').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendMessage();
            }
        });

        // Initialize page
        document.addEventListener('DOMContentLoaded', function() {
            checkAuth();
            loadConfig();
            checkAgentStatus();

            // Refresh agent status every 30 seconds
            setInterval(checkAgentStatus, 30000);
        });
    </script>
</body>
</html>`
