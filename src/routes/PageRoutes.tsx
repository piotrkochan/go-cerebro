import { useMemo } from 'react';
import { useNavigate, useSearch } from '@tanstack/react-router';
import { useStore } from '@tanstack/react-store';

import { AliasesPage } from '../pages/AliasesPage';
import { AnalysisPage } from '../pages/AnalysisPage';
import { CatPage } from '../pages/CatPage';
import { ClusterSettingsPage } from '../pages/ClusterSettingsPage';
import { ConnectPage } from '../pages/ConnectPage';
import { CreateIndexPage } from '../pages/CreateIndexPage';
import { IndexSettingsPage } from '../pages/IndexSettingsPage';
import { NodesPage } from '../pages/NodesPage';
import { OverviewPage } from '../pages/OverviewPage';
import { RepositoriesPage } from '../pages/RepositoriesPage';
import { RestPage } from '../pages/RestPage';
import { SnapshotPage } from '../pages/SnapshotPage';
import { TemplatesPage } from '../pages/TemplatesPage';
import { alertsActions } from '../stores/alertsStore';
import { refreshStore } from '../stores/refreshStore';
import { getConnection, sessionActions, sessionStore } from '../stores/sessionStore';

function usePageContext() {
  const session = useStore(sessionStore);
  const refreshTick = useStore(refreshStore, (state) => state.tick);
  const connection = useMemo(
    () => getConnection(session),
    [session.auth.password, session.auth.username, session.host],
  );

  return {
    connection,
    notify: alertsActions.notify,
    refreshTick,
    setStatus: sessionActions.setStatus,
  };
}

export function ConnectRoute() {
  const host = useStore(sessionStore, (state) => state.host);
  const navigate = useNavigate();
  return (
    <ConnectPage
      currentHost={host}
      onConnected={(nextHost, auth) => {
        sessionActions.connect(nextHost, auth);
        void navigate({ search: { host: nextHost }, to: '/overview' });
      }}
    />
  );
}

export function OverviewRoute() {
  const { connection, notify, refreshTick, setStatus } = usePageContext();
  return <OverviewPage connection={connection} notify={notify} refreshTick={refreshTick} setStatus={setStatus} />;
}

export function NodesRoute() {
  const { connection, refreshTick } = usePageContext();
  return <NodesPage connection={connection} refreshTick={refreshTick} />;
}

export function RestRoute() {
  const { connection } = usePageContext();
  return <RestPage connection={connection} />;
}

export function AliasesRoute() {
  const { connection, notify, refreshTick } = usePageContext();
  return <AliasesPage connection={connection} notify={notify} refreshTick={refreshTick} />;
}

export function RepositoriesRoute() {
  const { connection, notify, refreshTick } = usePageContext();
  return <RepositoriesPage connection={connection} notify={notify} refreshTick={refreshTick} />;
}

export function CatRoute() {
  const { connection, notify } = usePageContext();
  return <CatPage connection={connection} notify={notify} />;
}

export function AnalysisRoute() {
  const { connection, notify, refreshTick } = usePageContext();
  return <AnalysisPage connection={connection} notify={notify} refreshTick={refreshTick} />;
}

export function TemplatesRoute() {
  const { connection, notify, refreshTick } = usePageContext();
  return <TemplatesPage connection={connection} notify={notify} refreshTick={refreshTick} />;
}

export function SnapshotRoute() {
  const { connection, notify, refreshTick } = usePageContext();
  return <SnapshotPage connection={connection} notify={notify} refreshTick={refreshTick} />;
}

export function ClusterSettingsRoute() {
  const { connection, notify, refreshTick } = usePageContext();
  return <ClusterSettingsPage connection={connection} notify={notify} refreshTick={refreshTick} />;
}

export function CreateIndexRoute() {
  const { connection, notify, refreshTick } = usePageContext();
  return <CreateIndexPage connection={connection} notify={notify} refreshTick={refreshTick} />;
}

export function IndexSettingsRoute() {
  const { connection, notify } = usePageContext();
  const search = useSearch({ strict: false });
  return <IndexSettingsPage connection={connection} index={typeof search.index === 'string' ? search.index : ''} notify={notify} />;
}
