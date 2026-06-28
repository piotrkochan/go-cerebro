import { Outlet, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';
import { useStore } from '@tanstack/react-store';

import { Alerts } from '../components/Alerts';
import { Navbar } from '../components/Navbar';
import { navbar } from '../api/client';
import { alertsStore } from '../stores/alertsStore';
import { refreshActions, refreshStore } from '../stores/refreshStore';
import { getConnection, sessionActions, sessionStore, type ClusterHealthIssue } from '../stores/sessionStore';
import { textValue } from '../utils/format';

export function AppShell() {
  const alerts = useStore(alertsStore, (state) => state.alerts);
  const connected = useStore(sessionStore, (state) => state.connected);
  const healthIssue = useStore(sessionStore, (state) => state.healthIssue);
  const host = useStore(sessionStore, (state) => state.host);
  const status = useStore(sessionStore, (state) => state.status);
  const refreshInterval = useStore(refreshStore, (state) => state.interval);
  const refreshTick = useStore(refreshStore, (state) => state.tick);
  const navigate = useNavigate();

  useEffect(() => {
    if (!connected) void navigate({ to: '/connect' });
  }, [connected, navigate]);

  useEffect(() => {
    if (!connected) return;
    const timer = window.setInterval(() => refreshActions.tick(), refreshInterval);
    return () => window.clearInterval(timer);
  }, [connected, refreshInterval]);

  useEffect(() => {
    if (!connected || !host) return;
    let ignore = false;
    async function loadStatus() {
      try {
        const result = await navbar<true>({ body: getConnection(), throwOnError: true });
        if (!ignore) {
          const data = result.data.data as
            | { features?: { data_explorer?: unknown }; health_issue?: unknown; status?: unknown; version?: { number?: unknown } }
            | undefined;
          sessionActions.setFeatures({ dataExplorer: data?.features?.data_explorer === true });
          sessionActions.setHealthIssue(parseHealthIssue(data?.health_issue));
          sessionActions.setStatus(textValue(data?.status));
          sessionActions.setVersion(textValue(data?.version?.number));
        }
      } catch {
        if (!ignore) {
          sessionActions.setHealthIssue(null);
          sessionActions.setStatus('');
          sessionActions.setVersion('');
        }
      }
    }
    void loadStatus();
    return () => {
      ignore = true;
    };
  }, [connected, host, refreshTick]);

  function disconnect() {
    sessionActions.disconnect();
    void navigate({ to: '/connect' });
  }

  return (
    <>
      <Alerts alerts={alerts} />
      <Navbar
        connected={connected}
        connection={getConnection()}
        disconnect={disconnect}
        healthIssue={healthIssue}
        host={host}
        onHealthFixed={refreshActions.tick}
        refreshInterval={refreshInterval}
        setRefreshInterval={refreshActions.setInterval}
        status={status}
      />
      <div className="container-fluid main">
        <div className="row">
          <div className="col-sm-12">
            <div className="content">
              <Outlet />
            </div>
          </div>
        </div>
      </div>
    </>
  );
}

function parseHealthIssue(value: unknown): ClusterHealthIssue | null {
  if (!value || typeof value !== 'object') return null;
  const issue = value as Partial<ClusterHealthIssue>;
  const status = textValue(issue.status);
  if (status !== 'yellow' && status !== 'red') return null;
  return {
    fixes: Array.isArray(issue.fixes)
      ? issue.fixes.map((fix) => ({
          action: textValue(fix.action),
          index: textValue(fix.index),
          rationale: textValue(fix.rationale),
          setting: textValue(fix.setting),
          summary: textValue(fix.summary),
          value: textValue(fix.value),
        }))
      : [],
    status,
    summary: textValue(issue.summary) || `cluster health is ${status}`,
    unassigned_shard_count: Number(issue.unassigned_shard_count) || 0,
    unassigned_shards: Array.isArray(issue.unassigned_shards)
      ? issue.unassigned_shards.map((shard) => ({
          allocation_decision: textValue(shard.allocation_decision),
          deciders: Array.isArray(shard.deciders) ? shard.deciders.map(textValue).filter(Boolean) : [],
          explanation: textValue(shard.explanation),
          index: textValue(shard.index),
          primary_replica: textValue(shard.primary_replica),
          reason: textValue(shard.reason),
          shard: textValue(shard.shard),
        }))
      : [],
  };
}
