import { Outlet, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';
import { useStore } from '@tanstack/react-store';

import { Alerts } from '../components/Alerts';
import { Navbar } from '../components/Navbar';
import { navbar } from '../api/client';
import { alertsStore } from '../stores/alertsStore';
import { refreshActions, refreshStore } from '../stores/refreshStore';
import { getConnection, sessionActions, sessionStore } from '../stores/sessionStore';
import { textValue } from '../utils/format';

export function AppShell() {
  const alerts = useStore(alertsStore, (state) => state.alerts);
  const connected = useStore(sessionStore, (state) => state.connected);
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
          const data = result.data.data as { features?: { data_explorer?: unknown }; status?: unknown; version?: { number?: unknown } } | undefined;
          sessionActions.setFeatures({ dataExplorer: data?.features?.data_explorer === true });
          sessionActions.setStatus(textValue(data?.status));
          sessionActions.setVersion(textValue(data?.version?.number));
        }
      } catch {
        if (!ignore) {
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
        disconnect={disconnect}
        host={host}
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
