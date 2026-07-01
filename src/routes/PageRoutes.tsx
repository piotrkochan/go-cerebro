import { useCallback, useMemo } from 'react';
import { useNavigate, useSearch } from '@tanstack/react-router';
import { useStore } from '@tanstack/react-store';

import { AliasesPage } from '../pages/AliasesPage';
import { AnalysisPage } from '../pages/AnalysisPage';
import { CatPage } from '../pages/CatPage';
import { ClusterSettingsPage } from '../pages/ClusterSettingsPage';
import { ConnectPage } from '../pages/ConnectPage';
import { CreateIndexPage } from '../pages/CreateIndexPage';
import { DataExplorerPage } from '../pages/DataExplorerPage';
import { DataStreamsPage } from '../pages/DataStreamsPage';
import { ILMPoliciesPage } from '../pages/ILMPoliciesPage';
import { IndexSettingsPage } from '../pages/IndexSettingsPage';
import { LoginPage } from '../pages/LoginPage';
import { NodesPage } from '../pages/NodesPage';
import { OverviewPage } from '../pages/OverviewPage';
import { RepositoriesPage } from '../pages/RepositoriesPage';
import { RestPage } from '../pages/RestPage';
import { SnapshotPage } from '../pages/SnapshotPage';
import { TemplatesPage } from '../pages/TemplatesPage';
import { alertsActions } from '../stores/alertsStore';
import { refreshStore } from '../stores/refreshStore';
import { parseMajorVersion } from '../settingsCatalog';
import { getConnection, sessionActions, sessionStore } from '../stores/sessionStore';

function usePageContext() {
  const session = useStore(sessionStore);
  const refreshTick = useStore(refreshStore, (state) => state.tick);
  const connection = useMemo(
    () => getConnection(session),
    [session.host],
  );
  const notify = useCallback((kind: Parameters<typeof alertsActions.notify>[0], text: string) => {
    const current = sessionStore.state;
    if (!current.connected || current.host !== connection.host) return;
    alertsActions.notify(kind, text);
  }, [connection.host]);

  return {
    connection,
    majorVersion: parseMajorVersion(session.version),
    notify,
    refreshTick,
    setStatus: sessionActions.setStatus,
  };
}

export function ConnectRoute() {
  const savedHost = useStore(sessionStore, (state) => state.host);
  const search = useSearch({ strict: false });
  const host = typeof search.host === 'string' ? search.host : savedHost;
  const navigate = useNavigate();
  return (
    <ConnectPage
      currentHost={host}
      onConnected={(nextHost, nextHostName) => {
        sessionActions.connect(nextHost, nextHostName);
        void navigate({ search: { host: nextHost }, to: '/overview' });
      }}
    />
  );
}

export function LoginRoute() {
  return <LoginPage />;
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
  const { connection, majorVersion, notify, refreshTick } = usePageContext();
  const search = useSearch({ strict: false });
  const navigate = useNavigate();
  const updateKind = useCallback((kind: 'index' | 'component' | 'legacy') => {
    void navigate({ search: (previous) => ({ ...previous, kind, template: undefined }), to: '/templates' });
  }, [navigate]);
  return (
    <TemplatesPage
      connection={connection}
      majorVersion={majorVersion}
      notify={notify}
      refreshTick={refreshTick}
      selectedKind={search.kind === 'index' || search.kind === 'component' || search.kind === 'legacy' ? search.kind : undefined}
      selectedTemplate={typeof search.template === 'string' ? search.template : undefined}
      onKindChange={updateKind}
    />
  );
}

export function DataStreamsRoute() {
  const { connection, notify, refreshTick } = usePageContext();
  const search = useSearch({ strict: false });
  const navigate = useNavigate();
  return (
    <DataStreamsPage
      connection={connection}
      notify={notify}
      refreshTick={refreshTick}
      selectedStream={typeof search.stream === 'string' ? search.stream : ''}
      onStreamChange={(stream) => {
        void navigate({ search: (previous) => ({ ...previous, stream: stream || undefined }), to: '/data_streams' });
      }}
    />
  );
}

export function ILMPoliciesRoute() {
  const { connection, notify, refreshTick } = usePageContext();
  const search = useSearch({ strict: false });
  return <ILMPoliciesPage connection={connection} initialPolicy={typeof search.policy === 'string' ? search.policy : ''} notify={notify} refreshTick={refreshTick} />;
}

export function SnapshotRoute() {
  const { connection, notify, refreshTick } = usePageContext();
  return <SnapshotPage connection={connection} notify={notify} refreshTick={refreshTick} />;
}

export function ClusterSettingsRoute() {
  const { connection, majorVersion, notify, refreshTick } = usePageContext();
  return <ClusterSettingsPage connection={connection} majorVersion={majorVersion} notify={notify} refreshTick={refreshTick} />;
}

export function CreateIndexRoute() {
  const { connection, majorVersion, notify, refreshTick } = usePageContext();
  return <CreateIndexPage connection={connection} majorVersion={majorVersion} notify={notify} refreshTick={refreshTick} />;
}

export function IndexSettingsRoute() {
  const { connection, majorVersion, notify } = usePageContext();
  const search = useSearch({ strict: false });
  return <IndexSettingsPage connection={connection} index={typeof search.index === 'string' ? search.index : ''} majorVersion={majorVersion} notify={notify} />;
}

export function DataExplorerRoute() {
  const { connection, notify } = usePageContext();
  const enabled = useStore(sessionStore, (state) => state.features.dataExplorer);
  const search = useSearch({ strict: false });
  return (
    <DataExplorerPage
      connection={connection}
      enabled={enabled}
      index={typeof search.index === 'string' ? search.index : ''}
      notify={notify}
    />
  );
}
