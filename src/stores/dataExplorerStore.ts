import { createStore } from '@tanstack/react-store';

type DataExplorerState = {
  pageSize: number;
  queryMode: 'kql' | 'lucene';
  sortField: string;
  sortOrder: 'asc' | 'desc';
};

export const dataExplorerStore = createStore<DataExplorerState>({
  pageSize: savedPageSize(),
  queryMode: savedQueryMode(),
  sortField: '',
  sortOrder: 'asc',
});

export const dataExplorerActions = {
  setPageSize(pageSize: number) {
    window.localStorage.setItem('cerebro.dataExplorer.pageSize', String(pageSize));
    dataExplorerStore.setState((state) => ({ ...state, pageSize }));
  },
  setQueryMode(queryMode: DataExplorerState['queryMode']) {
    window.localStorage.setItem('cerebro.dataExplorer.queryMode', queryMode);
    dataExplorerStore.setState((state) => ({ ...state, queryMode }));
  },
  setSort(field: string) {
    dataExplorerStore.setState((state) => ({
      ...state,
      sortField: field,
      sortOrder: state.sortField === field && state.sortOrder === 'asc' ? 'desc' : 'asc',
    }));
  },
};

function savedPageSize() {
  const value = Number(window.localStorage.getItem('cerebro.dataExplorer.pageSize'));
  return [10, 25, 50, 100].includes(value) ? value : 25;
}

function savedQueryMode(): DataExplorerState['queryMode'] {
  const value = window.localStorage.getItem('cerebro.dataExplorer.queryMode');
  return value === 'lucene' ? 'lucene' : 'kql';
}
