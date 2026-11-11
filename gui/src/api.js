const DEFAULT_TOKEN = 'change_me_api_token';

// The session credential lives in memory only — it is never written to
// localStorage/sessionStorage (XSS-readable). The server converts it into an
// HttpOnly SameSite=Strict session cookie on the first authenticated request;
// afterwards fetch, WebSocket and iframe requests authenticate via that
// cookie automatically. A ?token= URL parameter seeds the in-memory token for
// this page load only (used to bootstrap a session on a new browser).
let sessionToken = (() => {
  try {
    const urlToken = new URLSearchParams(window.location.search).get('token');
    if (urlToken) return urlToken;
  } catch {
    // Non-browser/test environments: fall through to the default.
  }
  return DEFAULT_TOKEN;
})();

export const getToken = () => sessionToken;

export const setToken = (token) => {
  sessionToken = token || '';
};

export const getApiBase = () => {
  // If Vite dev server (port 5173), target the default Go API port (5050)
  if (window.location.port === '5173') {
    return 'http://localhost:5050';
  }
  return window.location.origin;
};

export const getWsBase = () => {
  const apiBase = getApiBase();
  return apiBase.replace(/^http/, 'ws');
};

const fetchAPI = async (method, path, body = null) => {
  const url = `${getApiBase()}${path}`;
  const headers = {
    'Content-Type': 'application/json',
  };
  if (sessionToken) {
    headers['X-API-Token'] = sessionToken;
  }

  const options = {
    method,
    headers,
  };

  if (body) {
    options.body = JSON.stringify(body);
  }

  // Bound every request: a hung connection would otherwise stall a polling
  // cycle forever. 30s is generous enough for plugin install/uninstall calls.
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 30000);
  options.signal = controller.signal;

  try {
    const res = await fetch(url, options);
    if (res.status === 401) {
      if (sessionToken === DEFAULT_TOKEN) {
        // The factory-default token was rejected — the HttpOnly session
        // cookie may still be valid. Drop the header and retry once so
        // cookie-based auth can proceed.
        sessionToken = '';
        return fetchAPI(method, path, body);
      }
      return { status: 'error', error: 'Unauthorized API Access' };
    }
    if (!res.ok) {
      const text = await res.text();
      return { status: 'error', error: text || `HTTP ${res.status}` };
    }
    const data = await res.json();
    if (data && data.status) {
      return data;
    }
    return { status: 'success', data };
  } catch (err) {
    if (err.name === 'AbortError') {
      return { status: 'error', error: 'Request timed out' };
    }
    return { status: 'error', error: err.message };
  } finally {
    clearTimeout(timeout);
  }
};

export const api = {
  getMetrics: () => fetchAPI('GET', '/api/v2/system/stats'),
  getSystemMetricsHistory: () => fetchAPI('GET', '/api/v2/system/metrics'),
  getSystemHealth: () => fetchAPI('GET', '/api/v2/system/health'),
  getPlugins: () => fetchAPI('GET', '/api/v2/plugins'),
  getPluginCatalog: () => fetchAPI('GET', '/api/v2/plugins/catalog'),
  getPluginChangelog: (name) => fetchAPI('GET', `/api/v2/plugins/${name}/changelog`),
  installPlugin: (name, payload = {}) => fetchAPI('POST', `/api/v2/plugins/${name}/install`, payload),
  uninstallPlugin: (name, payload = {}) => fetchAPI('POST', `/api/v2/plugins/${name}/uninstall`, payload),
  getPluginDownloadUrl: (name) => `${getApiBase()}/api/v2/plugins/${name}/download`,
  startPlugin: (name) => fetchAPI('POST', `/api/v2/plugins/${name}/start`),
  stopPlugin: (name) => fetchAPI('POST', `/api/v2/plugins/${name}/stop`),
  collectPlugin: (name) => fetchAPI('GET', `/api/v2/plugins/${name}/collect`),
  pluginAction: (name, action, args = {}) => fetchAPI('POST', `/api/v2/plugins/${name}/action`, { action, args }),
  getPluginTabs: () => fetchAPI('GET', '/api/v2/plugins/tabs'),
  getProcesses: (limit = 150) => fetchAPI('GET', `/api/v2/system/processes?limit=${limit}`),
  killProcess: (pid) => fetchAPI('POST', `/api/v2/system/processes/${pid}/kill`),
  getSystemSpecs: () => fetchAPI('GET', '/api/v2/system/specs'),
  getVersion: () => fetchAPI('GET', '/api/v2/version'),
  getSettings: () => fetchAPI('GET', '/api/v2/settings'),
  saveCoreSettings: (core) => fetchAPI('POST', '/api/v2/settings/core', core),
  getPluginSettings: (name) => fetchAPI('GET', `/api/v2/plugins/${name}/settings`),
  savePluginSettings: (name, config) => fetchAPI('POST', `/api/v2/plugins/${name}/settings`, { config }),
  getRepositories: () => fetchAPI('GET', '/api/v2/repositories'),
  saveRepository: (repo) => fetchAPI('POST', '/api/v2/repositories', repo),
  refreshRepositories: () => fetchAPI('POST', '/api/v2/repositories/refresh'),
  deleteRepository: (id) => fetchAPI('DELETE', `/api/v2/repositories/${id}`),
  getApiKeys: () => fetchAPI('GET', '/api/v2/auth/keys'),
  createApiKey: (name, payload = {}) => fetchAPI('POST', '/api/v2/auth/keys', { name, ...payload }),
  deleteApiKey: (id) => fetchAPI('DELETE', `/api/v2/auth/keys/${id}`),
};

export const subscribeWS = (path, onMessage, onError = null) => {
  // The session cookie is sent automatically on same-origin WebSocket
  // handshakes; browsers cannot attach headers to WS connections.
  const url = `${getWsBase()}${path}`;
  let ws = new WebSocket(url);
  let isClosed = false;

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      onMessage(data);
    } catch {
      // Handle raw text messages (like log strings)
      onMessage({ raw: true, data: event.data });
    }
  };

  ws.onerror = (err) => {
    if (onError) onError(err);
  };

  ws.onclose = () => {
    if (!isClosed) {
      // Reconnect after 3 seconds
      setTimeout(() => {
        if (!isClosed) {
          subscribeWS(path, onMessage, onError);
        }
      }, 3000);
    }
  };

  return {
    close: () => {
      isClosed = true;
      ws.close();
    }
  };
};
