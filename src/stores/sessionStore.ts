import type { HostBodyWritable } from '../api/client';
import { cleanConnection, savedHostKey, savedHostNameKey } from '../utils/connection';
import { createStore } from '@tanstack/react-store';

type SessionState = {
  connected: boolean;
  features: {
    dataExplorer: boolean;
  };
  healthIssue: ClusterHealthIssue | null;
  host: string;
  hostName: string;
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
const initialHostName = window.localStorage.getItem(savedHostNameKey) ?? initialHost;

export const sessionStore = createStore<SessionState>({
  connected: Boolean(initialHost),
  features: { dataExplorer: false },
  healthIssue: null,
  host: initialHost,
  hostName: initialHostName,
  status: '',
  version: '',
});

export const sessionActions = {
  connect(host: string, hostName = host) {
    window.localStorage.setItem(savedHostKey, host);
    window.localStorage.setItem(savedHostNameKey, hostName);
    sessionStore.setState((state) => ({ ...state, connected: true, healthIssue: null, host, hostName }));
  },
  disconnect() {
    window.localStorage.removeItem(savedHostKey);
    window.localStorage.removeItem(savedHostNameKey);
    sessionStore.setState(() => ({
      connected: false,
      features: { dataExplorer: false },
      healthIssue: null,
      host: '',
      hostName: '',
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
  setHostName(hostName: string) {
    window.localStorage.setItem(savedHostNameKey, hostName);
    sessionStore.setState((state) => (state.hostName === hostName ? state : { ...state, hostName }));
  },
  setStatus(status: string) {
    sessionStore.setState((state) => (state.status === status ? state : { ...state, status }));
  },
  setVersion(version: string) {
    sessionStore.setState((state) => (state.version === version ? state : { ...state, version }));
  },
};

export function getConnection(state = sessionStore.state): HostBodyWritable {
  return cleanConnection({ host: state.host });
}
