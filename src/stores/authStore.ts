import { createStore } from '@tanstack/react-store';

import type { AuthStatus } from '../api/security';

export type AuthPermission = {
  action: string;
  effect: string;
  object: string;
  resource: string;
};

type AuthState = {
  authenticated: boolean;
  enabled: boolean;
  groups: string[];
  loaded: boolean;
  permissions: AuthPermission[];
  provider: string;
  providerId: string;
  providers: {
    entraid: boolean;
    oauth: boolean;
    oauthName: string;
    password: boolean;
  };
  roles: string[];
  user: string;
};

const defaultState = (): AuthState => ({
  authenticated: false,
  enabled: false,
  groups: [],
  loaded: false,
  permissions: [],
  provider: '',
  providerId: '',
  providers: {
    entraid: false,
    oauth: false,
    oauthName: '',
    password: false,
  },
  roles: [],
  user: '',
});

export const authStore = createStore<AuthState>(defaultState());

export const authActions = {
  clear() {
    authStore.setState(() => defaultState());
  },
  setStatus(status: AuthStatus | null) {
    if (!status) {
      authStore.setState((state) => ({ ...state, loaded: true }));
      return;
    }
    authStore.setState(() => ({
      authenticated: status.authenticated === true,
      enabled: status.enabled === true,
      groups: list(status.groups),
      loaded: true,
      permissions: permissions(status.permissions),
      provider: text(status.provider),
      providerId: text(status.provider_id),
      providers: {
        entraid: status.providers?.entraid === true,
        oauth: status.providers?.oauth === true,
        oauthName: text(status.provider_names?.oauth || status.providers?.oauth_name),
        password: status.providers?.password === true,
      },
      roles: list(status.roles),
      user: text(status.user),
    }));
  },
};

function list(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : [];
}

function permissions(value: unknown): AuthPermission[] {
  if (!Array.isArray(value)) return [];
  return value
    .filter((item): item is Record<string, unknown> => item !== null && typeof item === 'object')
    .map((item) => ({
      action: text(item.action),
      effect: text(item.effect),
      object: text(item.object),
      resource: text(item.resource),
    }))
    .filter((item) => item.action || item.effect || item.object || item.resource);
}

function text(value: unknown): string {
  return typeof value === 'string' ? value : '';
}
