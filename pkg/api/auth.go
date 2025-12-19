package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/smallnest/langchat/pkg/auth"
	"github.com/smallnest/langchat/pkg/middleware"
)

// AuthAPI handles authentication related API endpoints
type AuthAPI struct {
	authService *auth.AuthService
	jwtAuth     *middleware.AuthMiddleware
}

// NewAuthAPI creates a new authentication API handler
func NewAuthAPI(authService *auth.AuthService, jwtAuth *middleware.AuthMiddleware) *AuthAPI {
	return &AuthAPI{
		authService: authService,
		jwtAuth:     jwtAuth,
	}
}

// RegisterRoutes registers authentication routes
func (a *AuthAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/auth/login", a.HandleLogin)
	mux.HandleFunc("/api/auth/register", a.HandleRegister)
	mux.HandleFunc("/api/auth/refresh", a.HandleRefresh)
	mux.HandleFunc("/api/auth/logout", a.HandleLogout)
	mux.HandleFunc("/api/auth/me", a.HandleGetCurrentUser)

	// Serve login page
	mux.HandleFunc("/login", a.HandleLoginPage)
	mux.HandleFunc("/register", a.HandleRegisterPage)
}

// HandleLogin handles user login
func (a *AuthAPI) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		a.serveLoginPage(w, r)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req auth.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := a.authService.Login(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Warning: Failed to encode login response: %v", err)
	}
}

// HandleRegister handles user registration
func (a *AuthAPI) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		a.serveRegisterPage(w, r)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req auth.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := a.authService.Register(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Warning: Failed to encode register response: %v", err)
	}
}

// HandleRefresh handles token refresh
func (a *AuthAPI) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := a.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Warning: Failed to encode refresh response: %v", err)
	}
}

// HandleLogout handles user logout
func (a *AuthAPI) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := a.authService.Logout(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"}); err != nil {
		log.Printf("Warning: Failed to encode logout response: %v", err)
	}
}

// HandleGetCurrentUser returns the current authenticated user
func (a *AuthAPI) HandleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from JWT token
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get full user details
	fullUser, exists := a.authService.GetUserByID(user.UserID)
	if !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&auth.UserInfo{
		ID:       fullUser.ID,
		Username: fullUser.Username,
		Email:    fullUser.Email,
		Roles:    fullUser.Roles,
	}); err != nil {
		log.Printf("Warning: Failed to encode user info response: %v", err)
	}
}

// serveLoginPage serves the login HTML page
func (a *AuthAPI) serveLoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(loginPageHTML)); err != nil {
		log.Printf("Warning: Failed to write login page HTML: %v", err)
	}
}

// serveRegisterPage serves the registration HTML page
func (a *AuthAPI) serveRegisterPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(registerPageHTML)); err != nil {
		log.Printf("Warning: Failed to write register page HTML: %v", err)
	}
}

// HandleLoginPage handles login page route
func (a *AuthAPI) HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	a.serveLoginPage(w, r)
}

// HandleRegisterPage handles registration page route
func (a *AuthAPI) HandleRegisterPage(w http.ResponseWriter, r *http.Request) {
	a.serveRegisterPage(w, r)
}

// HTML templates for login and register pages
const loginPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>登录 - 聊天智能体</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
            margin: 0;
            padding: 0;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
        }
        .login-container {
            background: white;
            padding: 2rem;
            border-radius: 10px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.15), 0 10px 25px rgba(0,0,0,0.1);
            width: 100%;
            max-width: 400px;
        }
        .login-header {
            text-align: center;
            margin-bottom: 2rem;
        }
        .login-header h1 {
            color: #333;
            margin-bottom: 0.5rem;
        }
        .login-header p {
            color: #666;
            margin: 0;
        }
        .form-group {
            margin-bottom: 1.5rem;
        }
        .form-group label {
            display: block;
            margin-bottom: 0.5rem;
            color: #333;
            font-weight: 500;
        }
        .form-group input {
            width: 100%;
            padding: 0.75rem;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-size: 1rem;
            box-sizing: border-box;
        }
        .form-group input:focus {
            outline: none;
            border-color: #00f2fe;
            box-shadow: 0 0 0 2px rgba(0, 242, 254, 0.1);
        }
        .login-button {
            width: 100%;
            padding: 0.75rem;
            background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
            color: white;
            border: none;
            border-radius: 8px;
            font-size: 1rem;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s ease;
            box-shadow: 0 4px 15px rgba(79, 172, 254, 0.3);
        }
        .login-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(79, 172, 254, 0.4);
        }
        .login-button:active {
            transform: translateY(0);
        }
        .login-footer {
            text-align: center;
            margin-top: 1.5rem;
            color: #666;
        }
        .login-footer a {
            color: #00f2fe;
            text-decoration: none;
            font-weight: 500;
        }
        .login-footer a:hover {
            color: #4facfe;
            text-decoration: underline;
        }
        .error-message {
            color: #e74c3c;
            margin-bottom: 1rem;
            padding: 0.75rem;
            background: #fdf2f2;
            border-radius: 5px;
            display: none;
        }
        .demo-info {
            background: #e8f4fd;
            border: 1px solid #bee5eb;
            border-radius: 5px;
            padding: 1rem;
            margin-bottom: 1.5rem;
            font-size: 0.9rem;
            color: #0c5460;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="login-header">
            <h1><img src="/static/images/logo.png" alt="聊天智能体" style="height: 40px; vertical-align: middle; margin-right: 10px;">聊天智能体</h1>
            <p>登录您的账号</p>
        </div>

        <div class="demo-info">
            <strong>演示账号:</strong><br>
            管理员: 用户名 <code>admin</code>, 密码 <code>admin123</code><br>
            普通用户: 用户名 <code>user</code>, 密码 <code>user123</code>
        </div>

        <div class="error-message" id="error-message"></div>

        <form id="login-form">
            <div class="form-group">
                <label for="username">用户名</label>
                <input type="text" id="username" name="username" required>
            </div>
            <div class="form-group">
                <label for="password">密码</label>
                <input type="password" id="password" name="password" required>
            </div>
            <button type="submit" class="login-button">登录</button>
        </form>

        <div class="login-footer">
            <p>还没有账号？<a href="/register">立即注册</a></p>
        </div>
    </div>

    <script>
        document.getElementById('login-form').addEventListener('submit', async (e) => {
            e.preventDefault();

            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            const errorDiv = document.getElementById('error-message');

            try {
                const response = await fetch('/api/auth/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ username, password })
                });

                const data = await response.json();

                if (response.ok) {
                    // Store tokens in localStorage (for API calls)
                    localStorage.setItem('access_token', data.access_token);
                    localStorage.setItem('refresh_token', data.refresh_token);
                    localStorage.setItem('user', JSON.stringify(data.user));

                    // Set cookie for browser requests
                    document.cookie = 'access_token=' + data.access_token + '; path=/; max-age=86400; SameSite=Lax';
                    document.cookie = 'refresh_token=' + data.refresh_token + '; path=/; max-age=604800; SameSite=Lax';

                    // Redirect to main app
                    window.location.href = '/';
                } else {
                    errorDiv.textContent = data.error || '登录失败';
                    errorDiv.style.display = 'block';
                }
            } catch (error) {
                errorDiv.textContent = '网络错误，请稍后重试。';
                errorDiv.style.display = 'block';
            }
        });
    </script>
</body>
</html>`

const registerPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>注册 - 聊天智能体</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
            margin: 0;
            padding: 0;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
        }
        .register-container {
            background: white;
            padding: 2rem;
            border-radius: 10px;
            box-shadow: 0 10px 25px rgba(0,0,0,0.1);
            width: 100%;
            max-width: 400px;
        }
        .register-header {
            text-align: center;
            margin-bottom: 2rem;
        }
        .register-header h1 {
            color: #333;
            margin-bottom: 0.5rem;
        }
        .register-header p {
            color: #666;
            margin: 0;
        }
        .form-group {
            margin-bottom: 1.5rem;
        }
        .form-group label {
            display: block;
            margin-bottom: 0.5rem;
            color: #333;
            font-weight: 500;
        }
        .form-group input {
            width: 100%;
            padding: 0.75rem;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-size: 1rem;
            box-sizing: border-box;
        }
        .form-group input:focus {
            outline: none;
            border-color: #00f2fe;
            box-shadow: 0 0 0 2px rgba(0, 242, 254, 0.1);
        }
        .register-button {
            width: 100%;
            padding: 0.75rem;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 5px;
            font-size: 1rem;
            font-weight: 500;
            cursor: pointer;
            transition: opacity 0.2s;
        }
        .register-button:hover {
            opacity: 0.9;
        }
        .register-footer {
            text-align: center;
            margin-top: 1.5rem;
            color: #666;
        }
        .register-footer a {
            color: #667eea;
            text-decoration: none;
        }
        .register-footer a:hover {
            text-decoration: underline;
        }
        .error-message {
            color: #e74c3c;
            margin-bottom: 1rem;
            padding: 0.75rem;
            background: #fdf2f2;
            border-radius: 5px;
            display: none;
        }
    </style>
</head>
<body>
    <div class="register-container">
        <div class="register-header">
            <h1><img src="/static/images/logo.png" alt="聊天智能体" style="height: 40px; vertical-align: middle; margin-right: 10px;">聊天智能体</h1>
            <p>创建您的账号</p>
        </div>

        <div class="error-message" id="error-message"></div>

        <form id="register-form">
            <div class="form-group">
                <label for="username">用户名</label>
                <input type="text" id="username" name="username" required>
            </div>
            <div class="form-group">
                <label for="email">电子邮箱</label>
                <input type="email" id="email" name="email" required>
            </div>
            <div class="form-group">
                <label for="password">密码</label>
                <input type="password" id="password" name="password" required minlength="6">
            </div>
            <button type="submit" class="register-button">注册</button>
        </form>

        <div class="register-footer">
            <p>已有账号？<a href="/login">立即登录</a></p>
        </div>
    </div>

    <script>
        document.getElementById('register-form').addEventListener('submit', async (e) => {
            e.preventDefault();

            const username = document.getElementById('username').value;
            const email = document.getElementById('email').value;
            const password = document.getElementById('password').value;
            const errorDiv = document.getElementById('error-message');

            try {
                const response = await fetch('/api/auth/register', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ username, email, password })
                });

                const data = await response.json();

                if (response.ok) {
                    // Store tokens in localStorage (for API calls)
                    localStorage.setItem('access_token', data.access_token);
                    localStorage.setItem('refresh_token', data.refresh_token);
                    localStorage.setItem('user', JSON.stringify(data.user));

                    // Set cookie for browser requests
                    document.cookie = 'access_token=' + data.access_token + '; path=/; max-age=86400; SameSite=Lax';
                    document.cookie = 'refresh_token=' + data.refresh_token + '; path=/; max-age=604800; SameSite=Lax';

                    // Redirect to main app
                    window.location.href = '/';
                } else {
                    errorDiv.textContent = data.error || '注册失败';
                    errorDiv.style.display = 'block';
                }
            } catch (error) {
                errorDiv.textContent = '网络错误，请稍后重试。';
                errorDiv.style.display = 'block';
            }
        });
    </script>
</body>
</html>`
