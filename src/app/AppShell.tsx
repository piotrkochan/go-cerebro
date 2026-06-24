import { Outlet, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';
import { useStore } from '@tanstack/react-store';

import { Alerts } from '../components/Alerts';
import { Navbar } from '../components/Navbar';
import { alertsStore } from '../stores/alertsStore';
import { refreshActions, refreshStore } from '../stores/refreshStore';
import { sessionActions, sessionStore } from '../stores/sessionStore';

export function AppShell() {
  const alerts = useStore(alertsStore, (state) => state.alerts);
  const connected = useStore(sessionStore, (state) => state.connected);
  const host = useStore(sessionStore, (state) => state.host);
  const status = useStore(sessionStore, (state) => state.status);
  const refreshInterval = useStore(refreshStore, (state) => state.interval);
  const navigate = useNavigate();

  useEffect(() => {
    if (!connected) void navigate({ to: '/connect' });
  }, [connected, navigate]);

  useEffect(() => {
    if (!connected) return;
    const timer = window.setInterval(() => refreshActions.tick(), refreshInterval);
    return () => window.clearInterval(timer);
  }, [connected, refreshInterval]);

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
