import { client } from './client/client.gen';
import { authActions, authStore, type AuthPermission } from '../stores/authStore';

export type AuthStatus = {
  authenticated?: boolean;
  csrf_token?: string;
  enabled?: boolean;
  groups?: string[];
  permissions?: AuthPermission[];
  provider?: string;
  providers?: {
    entraid?: boolean;
    oauth?: boolean;
    oauth_name?: string;
    password?: boolean;
  };
  provider_names?: {
    oauth?: string;
  };
  roles?: string[];
  user?: string;
};

let csrfToken = '';
let csrfLoaded = false;
let csrfPromise: Promise<AuthStatus | null> | null = null;
let authStatusCache: AuthStatus | null = null;
let authStatusLoadedAt = 0;
let authStatusPromise: Promise<AuthStatus | null> | null = null;

const AUTH_STATUS_CACHE_MS = 1000;

export function configureAPIClientSecurity() {
  client.setConfig({ baseUrl: apiBasePath() });
  client.interceptors.request.use(async (request) => {
    if (!csrfToken) await ensureCSRFToken();
    if (csrfToken) request.headers.set('X-Cerebro-CSRF', csrfToken);
    return request;
  });
}

export function apiURL(path: string): string {
  return `${apiBasePath()}${path.startsWith('/') ? path : `/${path}`}`;
}

export function setCSRFToken(token: unknown) {
  csrfToken = typeof token === 'string' ? token : '';
  csrfLoaded = true;
}

export function clearAuthSecurityState() {
  csrfToken = '';
  csrfLoaded = false;
  csrfPromise = null;
  authStatusCache = null;
  authStatusLoadedAt = 0;
  authStatusPromise = null;
  authActions.clear();
}

export async function loadAuthStatus(): Promise<AuthStatus | null> {
  const now = Date.now();
  if (authStatusCache && now - authStatusLoadedAt < AUTH_STATUS_CACHE_MS) {
    return authStatusCache;
  }
  if (!authStatusPromise) {
    authStatusPromise = fetchAuthStatus().finally(() => {
      authStatusPromise = null;
    });
  }
  const status = await authStatusPromise;
  setCSRFToken(status?.csrf_token);
  authStatusCache = status;
  authStatusLoadedAt = Date.now();
  authActions.setStatus(status);
  return status;
}

export async function loadAuthMe(): Promise<AuthStatus> {
  const response = await fetch(apiURL('/auth/me'), { headers: { Accept: 'application/json' } });
  if (!response.ok) {
    throw new Error(response.status === 401 ? 'authentication required' : 'unable to load profile');
  }
  const status = await response.json() as AuthStatus;
  authActions.setStatus({ ...status, authenticated: true, enabled: true });
  return status;
}

export async function logout(): Promise<void> {
  const proxyLogout = authStore.state.provider === 'proxy';
  try {
    if (!csrfToken) await ensureCSRFToken();
    const headers = new Headers({ Accept: 'application/json' });
    if (csrfToken) headers.set('X-Cerebro-CSRF', csrfToken);
    await fetch(apiURL('/auth/logout'), {
      credentials: 'same-origin',
      headers,
      method: 'POST',
    });
  } finally {
    clearAuthSecurityState();
    if (proxyLogout) {
      window.location.assign(`/oauth2/sign_out?rd=${encodeURIComponent('/oauth2/sign_in')}`);
    }
  }
}

async function ensureCSRFToken(): Promise<AuthStatus | null> {
  if (csrfToken || csrfLoaded) return null;
  if (!csrfPromise) {
    csrfPromise = loadAuthStatus().finally(() => {
      csrfPromise = null;
    });
  }
  return csrfPromise;
}

async function fetchAuthStatus(): Promise<AuthStatus | null> {
  try {
    const response = await fetch(apiURL('/auth/status'), { headers: { Accept: 'application/json' } });
    if (!response.ok) {
      authActions.setStatus(null);
      return null;
    }
    return await response.json() as AuthStatus;
  } catch {
    authActions.setStatus(null);
    return null;
  }
}

function apiBasePath(): string {
  const pathname = window.location.pathname;
  if (!pathname || pathname === '/') return '';
  return pathname.endsWith('/') ? pathname.slice(0, -1) : pathname;
}
