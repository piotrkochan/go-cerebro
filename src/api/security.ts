import { client } from './client/client.gen';

type AuthStatus = {
  authenticated?: boolean;
  csrf_token?: string;
  enabled?: boolean;
  user?: string;
};

let csrfToken = '';
let csrfLoaded = false;
let csrfPromise: Promise<AuthStatus | null> | null = null;

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

export async function loadAuthStatus(): Promise<AuthStatus | null> {
  const status = await fetchAuthStatus();
  setCSRFToken(status?.csrf_token);
  return status;
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
    if (!response.ok) return null;
    return await response.json() as AuthStatus;
  } catch {
    return null;
  }
}
