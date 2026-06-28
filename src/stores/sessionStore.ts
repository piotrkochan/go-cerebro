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
  healthIssue: ClusterHealthIssue | null;
  host: string;
  status: string;
  version: string;
};

export type ClusterHealthIssue = {
  fixes: ClusterHealthFix[];
  status: string;
  summary: string;
  unassigned_shard_count: number;
  unassigned_shards: ClusterUnassignedShard[];
};

export type ClusterHealthFix = {
  action: string;
  index: string;
  rationale: string;
  setting: string;
  summary: string;
  value: string;
};

export type ClusterUnassignedShard = {
  allocation_decision: string;
  deciders: string[];
  explanation: string;
  index: string;
  primary_replica: string;
  reason: string;
  shard: string;
};

const initialHost = window.localStorage.getItem(savedHostKey) ?? '';

export const sessionStore = createStore<SessionState>({
  auth: {},
  connected: Boolean(initialHost),
  features: { dataExplorer: false },
  healthIssue: null,
  host: initialHost,
  status: '',
  version: '',
});

export const sessionActions = {
  connect(host: string, auth: ConnectionAuth = {}) {
    window.localStorage.setItem(savedHostKey, host);
    sessionStore.setState((state) => ({ ...state, auth, connected: true, healthIssue: null, host }));
  },
  disconnect() {
    window.localStorage.removeItem(savedHostKey);
    sessionStore.setState(() => ({
      auth: {},
      connected: false,
      features: { dataExplorer: false },
      healthIssue: null,
      host: '',
      status: '',
      version: '',
    }));
  },
  setFeatures(features: Partial<SessionState['features']>) {
    sessionStore.setState((state) => ({ ...state, features: { ...state.features, ...features } }));
  },
  setHealthIssue(healthIssue: ClusterHealthIssue | null) {
    sessionStore.setState((state) => ({ ...state, healthIssue }));
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
