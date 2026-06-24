import {
  createHashHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Navigate,
  redirect,
} from '@tanstack/react-router';
import type { ReactNode } from 'react';

import { AppShell } from './AppShell';
import {
  AliasesRoute,
  AnalysisRoute,
  CatRoute,
  ClusterSettingsRoute,
  ConnectRoute,
  CreateIndexRoute,
  DataExplorerRoute,
  IndexSettingsRoute,
  NodesRoute,
  OverviewRoute,
  RepositoriesRoute,
  RestRoute,
  SnapshotRoute,
  TemplatesRoute,
} from '../routes/PageRoutes';
import { sessionActions, sessionStore } from '../stores/sessionStore';

type AppSearch = {
  host?: string;
  index?: string;
};

const validateSearch = (search: Record<string, unknown>): AppSearch => ({
  host: typeof search.host === 'string' ? search.host : undefined,
  index: typeof search.index === 'string' ? search.index : undefined,
});

const rootRoute = createRootRoute({
  component: AppShell,
});

const connectRoute = createRoute({
  component: ConnectRoute,
  getParentRoute: () => rootRoute,
  path: '/connect',
  validateSearch,
});

function appRoute<TPath extends string>(path: TPath, component: () => ReactNode) {
  return createRoute({
    beforeLoad: ({ search }) => {
      if (search.host && search.host !== sessionStore.state.host) {
        sessionActions.connect(search.host);
      }
      if (!sessionStore.state.connected) {
        throw redirect({ search: search.host ? { host: search.host } : {}, to: '/connect' });
      }
    },
    component,
    getParentRoute: () => rootRoute,
    path,
    validateSearch,
  });
}

const indexRoute = createRoute({
  component: () => <Navigate to={sessionStore.state.connected ? '/overview' : '/connect'} />,
  getParentRoute: () => rootRoute,
  path: '/',
});

const overviewRoute = appRoute('/overview', OverviewRoute);
const nodesRoute = appRoute('/nodes', NodesRoute);
const restRoute = appRoute('/rest', RestRoute);
const aliasesRoute = appRoute('/aliases', AliasesRoute);
const repositoriesRoute = appRoute('/repositories', RepositoriesRoute);
const repositoryRoute = appRoute('/repository', RepositoriesRoute);
const catRoute = appRoute('/cat', CatRoute);
const analysisRoute = appRoute('/analysis', AnalysisRoute);
const templatesRoute = appRoute('/templates', TemplatesRoute);
const snapshotRoute = appRoute('/snapshot', SnapshotRoute);
const clusterSettingsRoute = appRoute('/cluster_settings', ClusterSettingsRoute);
const createRoutePage = appRoute('/create', CreateIndexRoute);
const indexSettingsRoute = appRoute('/index_settings', IndexSettingsRoute);
const dataExplorerRoute = appRoute('/data_explorer', DataExplorerRoute);

const routeTree = rootRoute.addChildren([
  indexRoute,
  connectRoute,
  overviewRoute,
  nodesRoute,
  restRoute,
  aliasesRoute,
  repositoriesRoute,
  repositoryRoute,
  catRoute,
  analysisRoute,
  templatesRoute,
  snapshotRoute,
  clusterSettingsRoute,
  createRoutePage,
  indexSettingsRoute,
  dataExplorerRoute,
]);

export const router = createRouter({
  history: createHashHistory(),
  routeTree,
});

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
