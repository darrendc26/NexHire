// NexHire Auth Test Dashboard Controller
document.addEventListener('DOMContentLoaded', () => {
  // Config
  const API_BASE = window.location.origin.includes('localhost') || window.location.origin.includes('127.0.0.1')
    ? window.location.origin
    : 'http://localhost:8080';

  // DOM Elements
  const statusBadge = document.getElementById('status-badge');
  const statusText = document.getElementById('status-text');

  const btnOAuthLogin = document.getElementById('btn-oauth-login');
  const btnVerifyToken = document.getElementById('btn-verify-token');
  const inputIDToken = document.getElementById('input-id-token');

  const btnFetchMe = document.getElementById('btn-fetch-me');
  const btnFetchProtected = document.getElementById('btn-fetch-protected');
  const btnLogout = document.getElementById('btn-logout');

  const profileEmpty = document.getElementById('profile-empty');
  const profileDetails = document.getElementById('profile-details');
  const userAvatar = document.getElementById('user-avatar');
  const userName = document.getElementById('user-name');
  const userEmail = document.getElementById('user-email');
  const userID = document.getElementById('user-id');
  const userGoogleID = document.getElementById('user-google-id');
  const userCreatedAt = document.getElementById('user-created-at');
  const userRoleBadge = document.getElementById('user-role-badge');

  const tokenDisplay = document.getElementById('token-display');
  const btnCopyToken = document.getElementById('btn-copy-token');
  const tokenStatusPill = document.getElementById('token-status-pill');
  const decodedJwtSection = document.getElementById('decoded-jwt-section');
  const decodedTokenPayload = document.getElementById('decoded-token-payload');

  const consoleLogs = document.getElementById('console-logs');
  const btnClearLogs = document.getElementById('btn-clear-logs');
  const toastContainer = document.getElementById('toast-container');

  let currentToken = null;

  // Initialize
  init();

  async function init() {
    setupEventListeners();
    checkURLParameters();
    await checkAuthState();
  }

  function setupEventListeners() {
    btnOAuthLogin.addEventListener('click', handleOAuthLogin);
    btnVerifyToken.addEventListener('click', handleVerifyIDToken);
    btnFetchMe.addEventListener('click', () => fetchMeEndpoint(true));
    btnFetchProtected.addEventListener('click', fetchProtectedEndpoint);
    btnLogout.addEventListener('click', handleLogout);
    btnCopyToken.addEventListener('click', copyTokenToClipboard);
    btnClearLogs.addEventListener('click', clearLogs);
  }

  // Check URL parameters for OAuth redirect token
  function checkURLParameters() {
    const urlParams = new URLSearchParams(window.location.search);
    const token = urlParams.get('token');
    if (token) {
      logToConsole('INFO', 'Extracted JWT token from redirect URL parameter.');
      setToken(token);
      showToast('OAuth login callback received!', 'success');
      // Clean query parameter from URL without reload
      window.history.replaceState({}, document.title, window.location.pathname);
    }
  }

  // Handle Google OAuth Redirect Login
  function handleOAuthLogin() {
    logToConsole('INFO', `Redirecting to Google OAuth endpoint: ${API_BASE}/api/auth/google/login`);
    window.location.href = `${API_BASE}/api/auth/google/login`;
  }

  // Handle Direct ID Token Verification
  async function handleVerifyIDToken() {
    const idToken = inputIDToken.value.trim();
    if (!idToken) {
      showToast('Please paste a Google ID Token first', 'error');
      return;
    }

    logToConsole('INFO', `Sending POST request to ${API_BASE}/api/auth/google/verify`);

    try {
      const response = await fetch(`${API_BASE}/api/auth/google/verify`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ id_token: idToken }),
      });

      const data = await response.json();
      logToConsole(response.ok ? 'SUCCESS' : 'ERROR', `POST /api/auth/google/verify (${response.status})`, data);

      if (response.ok) {
        if (data.token) {
          setToken(data.token);
        }
        if (data.user) {
          updateProfileUI(data.user);
        }
        showToast('Google ID Token verified successfully!', 'success');
      } else {
        showToast(data.error || 'ID Token verification failed', 'error');
      }
    } catch (err) {
      logToConsole('ERROR', 'Failed to communicate with auth backend server.', err.message);
      showToast('Network error verifying token', 'error');
    }
  }

  // Fetch Current Auth State / GET /api/auth/me
  async function checkAuthState() {
    await fetchMeEndpoint(false);
  }

  async function fetchMeEndpoint(showToastNotification = true) {
    logToConsole('INFO', `Fetching current user via GET ${API_BASE}/api/auth/me`);

    const headers = {};
    if (currentToken) {
      headers['Authorization'] = `Bearer ${currentToken}`;
    }

    try {
      const response = await fetch(`${API_BASE}/api/auth/me`, {
        method: 'GET',
        headers,
        credentials: 'include',
      });

      if (response.ok) {
        const user = await response.json();
        logToConsole('SUCCESS', `GET /api/auth/me (${response.status})`, user);
        updateProfileUI(user);
        if (showToastNotification) showToast('User profile fetched successfully!', 'success');
      } else {
        const text = await response.text();
        logToConsole('WARNING', `GET /api/auth/me (${response.status}): ${text}`);
        updateUnauthenticatedUI();
        if (showToastNotification) showToast('Session unauthenticated or expired', 'error');
      }
    } catch (err) {
      logToConsole('ERROR', 'GET /api/auth/me failed (Network Error)', err.message);
      updateUnauthenticatedUI();
    }
  }

  // Fetch Protected Route
  async function fetchProtectedEndpoint() {
    logToConsole('INFO', `Testing protected route GET ${API_BASE}/api/protected/profile`);

    const headers = {};
    if (currentToken) {
      headers['Authorization'] = `Bearer ${currentToken}`;
    }

    try {
      const response = await fetch(`${API_BASE}/api/protected/profile`, {
        method: 'GET',
        headers,
        credentials: 'include',
      });

      const data = await response.json();
      logToConsole(response.ok ? 'SUCCESS' : 'ERROR', `GET /api/protected/profile (${response.status})`, data);

      if (response.ok) {
        showToast('Protected Endpoint Access Granted!', 'success');
      } else {
        showToast(`Protected Endpoint Denied (${response.status})`, 'error');
      }
    } catch (err) {
      logToConsole('ERROR', 'GET /api/protected/profile failed', err.message);
      showToast('Network error calling protected API', 'error');
    }
  }

  // Logout Handler
  async function handleLogout() {
    logToConsole('INFO', `Sending POST request to ${API_BASE}/api/auth/logout`);

    try {
      const response = await fetch(`${API_BASE}/api/auth/logout`, {
        method: 'POST',
        credentials: 'include',
      });

      const data = await response.json();
      logToConsole('SUCCESS', `POST /api/auth/logout (${response.status})`, data);

      currentToken = null;
      tokenDisplay.value = '';
      tokenStatusPill.className = 'pill pill-inactive';
      tokenStatusPill.textContent = 'No Token';
      decodedJwtSection.classList.add('hidden');

      updateUnauthenticatedUI();
      showToast('Logged out successfully', 'success');
    } catch (err) {
      logToConsole('ERROR', 'Logout failed', err.message);
      showToast('Error executing logout', 'error');
    }
  }

  // UI State Updaters
  function updateProfileUI(user) {
    statusBadge.className = 'status-badge authenticated';
    statusText.textContent = 'Authenticated';

    profileEmpty.classList.add('hidden');
    profileDetails.classList.remove('hidden');

    userAvatar.src = user.picture || 'https://via.placeholder.com/150';
    userName.textContent = user.name || 'Anonymous User';
    userEmail.textContent = user.email || 'No email provided';
    userID.textContent = user.id || 'N/A';
    userGoogleID.textContent = user.google_id || 'N/A';
    
    if (user.created_at) {
      const date = new Date(user.created_at);
      userCreatedAt.textContent = date.toLocaleString();
    } else {
      userCreatedAt.textContent = 'Just now';
    }

    btnLogout.removeAttribute('disabled');
  }

  function updateUnauthenticatedUI() {
    statusBadge.className = 'status-badge unauthenticated';
    statusText.textContent = 'Unauthenticated';

    profileDetails.classList.add('hidden');
    profileEmpty.classList.remove('hidden');

    btnLogout.setAttribute('disabled', 'true');
  }

  function setToken(tokenStr) {
    currentToken = tokenStr;
    tokenDisplay.value = tokenStr;
    tokenStatusPill.className = 'pill pill-active';
    tokenStatusPill.textContent = 'Active JWT';

    try {
      const payload = parseJwt(tokenStr);
      decodedTokenPayload.textContent = JSON.stringify(payload, null, 2);
      decodedJwtSection.classList.remove('hidden');
    } catch (e) {
      logToConsole('WARNING', 'Failed to decode JWT string format');
    }
  }

  function parseJwt(token) {
    const base64Url = token.split('.')[1];
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
    const jsonPayload = decodeURIComponent(window.atob(base64).split('').map(function(c) {
        return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
    }).join(''));

    return JSON.parse(jsonPayload);
  }

  function copyTokenToClipboard() {
    if (!tokenDisplay.value) return;
    navigator.clipboard.writeText(tokenDisplay.value);
    showToast('JWT copied to clipboard!', 'success');
  }

  // Console Logging Helper
  function logToConsole(type, message, detail = null) {
    const entry = document.createElement('div');
    entry.className = `log-entry log-${type.toLowerCase()}`;

    const timestamp = new Date().toLocaleTimeString();
    let content = `<span class="log-time">[${timestamp}]</span> <strong>${escapeHTML(message)}</strong>`;

    if (detail) {
      if (typeof detail === 'object') {
        content += `<br><span style="opacity: 0.85; white-space: pre-wrap;">${escapeHTML(JSON.stringify(detail, null, 2))}</span>`;
      } else {
        content += `<br><span style="opacity: 0.85;">${escapeHTML(detail)}</span>`;
      }
    }

    entry.innerHTML = content;
    consoleLogs.appendChild(entry);
    consoleLogs.scrollTop = consoleLogs.scrollHeight;
  }

  function clearLogs() {
    consoleLogs.innerHTML = '';
    logToConsole('INFO', 'Console logs cleared.');
  }

  function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    toastContainer.appendChild(toast);

    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transform = 'translateX(100%)';
      toast.style.transition = 'all 0.3s ease';
      setTimeout(() => toast.remove(), 300);
    }, 3500);
  }

  function escapeHTML(str) {
    if (typeof str !== 'string') return str;
    return str.replace(/[&<>'"]/g, 
      tag => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[tag] || tag)
    );
  }
});
