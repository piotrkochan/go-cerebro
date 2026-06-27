import type { HostBodyWritable } from '../api/client';
import type { ConnectionAuth } from '../types';
import { cleanConnection, savedHostKey } from '../utils/connection';
import { createStore } from '@tanstack/react-store';

type SessionState = {
  auth: ConnectionAuth;
  connected: boolean;
  features: {
    dataExplorer: boolean;
  };
  host: string;
  status: string;
  version: string;
};

const initialHost = window.localStorage.getItem(savedHostKey) ?? '';

export const sessionStore = createStore<SessionState>({
  auth: {},
  connected: Boolean(initialHost),
  features: { dataExplorer: false },
  host: initialHost,
  status: '',
  version: '',
});

export const sessionActions = {
  connect(host: string, auth: ConnectionAuth = {}) {
    window.localStorage.setItem(savedHostKey, host);
    sessionStore.setState((state) => ({ ...state, auth, connected: true, host }));
  },
  disconnect() {
    window.localStorage.removeItem(savedHostKey);
    sessionStore.setState(() => ({ auth: {}, connected: false, features: { dataExplorer: false }, host: '', status: '', version: '' }));
  },
  setFeatures(features: Partial<SessionState['features']>) {
    sessionStore.setState((state) => ({ ...state, features: { ...state.features, ...features } }));
  },
  setStatus(status: string) {
    sessionStore.setState((state) => (state.status === status ? state : { ...state, status }));
  },
  setVersion(version: string) {
    sessionStore.setState((state) => (state.version === version ? state : { ...state, version }));
  },
};

export function getConnection(state = sessionStore.state): HostBodyWritable {
  return cleanConnection({ host: state.host, ...state.auth });
}
