const DEFAULT_TOKEN = 'change_me_api_token';

export const getToken = () => {
  if (typeof window !== 'undefined' && window.location) {
    const params = new URLSearchParams(window.location.search);
    const urlToken = params.get('token');
    if (urlToken) {
      setToken(urlToken);
      return urlToken;
    }
  }
  return localStorage.getItem('m3tal_token') || DEFAULT_TOKEN;
};

export const setToken = (token) => {
  localStorage.setItem('m3tal_token', token);
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
    'X-API-Token': getToken(),
  };

  const options = {
    method,
    headers,
  };

  if (body) {
    options.body = JSON.stringify(body);
  }

  try {
    const res = await fetch(url, options);
    if (res.status === 401) {
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
    return { status: 'error', error: err.message };
  }
};

export const api = {
  getStacks: () => fetchAPI('GET', '/api/v2/stacks'),
  deployStack: (name) => fetchAPI('POST', `/api/v2/stacks/${name}/up`),
  stopStack: (name) => fetchAPI('POST', `/api/v2/stacks/${name}/down`),
  getContainers: () => fetchAPI('GET', '/api/v2/services'),
  controlContainer: (name, action) => fetchAPI('POST', `/api/v2/containers/${name}/${action}`),
  getMetrics: () => fetchAPI('GET', '/api/v2/system/stats'),
  getSystemMetricsHistory: () => fetchAPI('GET', '/api/v2/system/metrics'),
  getSystemHealth: () => fetchAPI('GET', '/api/v2/system/health'),
  getPlugins: () => fetchAPI('GET', '/api/v2/plugins'),
  getPluginCatalog: () => fetchAPI('GET', '/api/v2/plugins/catalog'),
  getPluginChangelog: (name) => fetchAPI('GET', `/api/v2/plugins/${name}/changelog`),
  installPlugin: (name, payload = {}) => fetchAPI('POST', `/api/v2/plugins/${name}/install`, payload),
  uninstallPlugin: (name, payload = {}) => fetchAPI('POST', `/api/v2/plugins/${name}/uninstall`, payload),
  getPluginDownloadUrl: (name) => `${getApiBase()}/api/v2/plugins/${name}/download?token=${getToken()}`,
  startPlugin: (name) => fetchAPI('POST', `/api/v2/plugins/${name}/start`),
  stopPlugin: (name) => fetchAPI('POST', `/api/v2/plugins/${name}/stop`),
  collectPlugin: (name) => fetchAPI('GET', `/api/v2/plugins/${name}/collect`),
  pluginAction: (name, action, args = {}) => fetchAPI('POST', `/api/v2/plugins/${name}/action`, { action, args }),
  getPluginTabs: () => fetchAPI('GET', '/api/v2/plugins/tabs'),
  runDoctor: () => fetchAPI('GET', '/api/v2/system/doctor'),
  pruneImages: () => fetchAPI('POST', '/api/v2/docker/images/prune'),
  systemCleanup: () => fetchAPI('POST', '/api/v2/system/cleanup'),
  reconcileSystem: () => fetchAPI('POST', '/api/v2/system/reconcile'),
  getVPNStatus: () => fetchAPI('GET', '/api/v2/vpn/status'),
  getRoutes: () => fetchAPI('GET', '/api/v2/routes'),
  getProcesses: (limit = 150) => fetchAPI('GET', `/api/v2/system/processes?limit=${limit}`),
  killProcess: (pid) => fetchAPI('POST', `/api/v2/system/processes/${pid}/kill`),
  getSystemSpecs: () => fetchAPI('GET', '/api/v2/system/specs'),
};

export const subscribeWS = (path, onMessage, onError = null) => {
  const token = getToken();
  const url = `${getWsBase()}${path}?token=${token}`;
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
