import type { HostBodyWritable } from '../api/client';
import type { ConnectionAuth } from '../types';
import { cleanConnection, savedHostKey } from '../utils/connection';
import { createStore } from '@tanstack/react-store';

type SessionState = {
  auth: ConnectionAuth;
  connected: boolean;
  host: string;
  status: string;
};

const initialHost = window.localStorage.getItem(savedHostKey) ?? '';

export const sessionStore = createStore<SessionState>({
  auth: {},
  connected: Boolean(initialHost),
  host: initialHost,
  status: '',
});

export const sessionActions = {
  connect(host: string, auth: ConnectionAuth = {}) {
    window.localStorage.setItem(savedHostKey, host);
    sessionStore.setState((state) => ({ ...state, auth, connected: true, host }));
  },
  disconnect() {
    window.localStorage.removeItem(savedHostKey);
    sessionStore.setState(() => ({ auth: {}, connected: false, host: '', status: '' }));
  },
  setStatus(status: string) {
    sessionStore.setState((state) => (state.status === status ? state : { ...state, status }));
  },
};

export function getConnection(state = sessionStore.state): HostBodyWritable {
  return cleanConnection({ host: state.host, ...state.auth });
}
