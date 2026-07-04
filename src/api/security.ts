import { client } from './client/client.gen';
import { authActions, type AuthPermission } from '../stores/authStore';

export type AuthStatus = {
  authenticated?: boolean;
  csrf_token?: string;
  enabled?: boolean;
  groups?: string[];
  permissions?: AuthPermission[];
  provider?: string;
  providers?: {
    entraid?: boolean;
    password?: boolean;
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
  client.interceptors.request.use(async (request) => {
    if (!csrfToken) await ensureCSRFToken();
    if (csrfToken) request.headers.set('X-Cerebro-CSRF', csrfToken);
    return request;
  });
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
  const response = await fetch('/auth/me', { headers: { Accept: 'application/json' } });
  if (!response.ok) {
    throw new Error(response.status === 401 ? 'authentication required' : 'unable to load profile');
  }
  const status = await response.json() as AuthStatus;
  authActions.setStatus({ ...status, authenticated: true, enabled: true });
  return status;
}

export async function logout(): Promise<void> {
  try {
    await fetch('/auth/logout', {
      credentials: 'same-origin',
      headers: { Accept: 'application/json' },
      method: 'POST',
    });
  } finally {
    clearAuthSecurityState();
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
    const response = await fetch('/auth/status', { headers: { Accept: 'application/json' } });
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
