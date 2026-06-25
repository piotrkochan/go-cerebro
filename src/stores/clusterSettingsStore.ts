import { createStore } from '@tanstack/react-store';

export type ClusterSettingChange = { transient: boolean; value: string };

type ClusterSettingsFilter = {
  name: string;
  showStatic: boolean;
};

type ClusterSettingsState = {
  hostKey: string;
  settings: Record<string, string>;
  form: Record<string, string>;
  changes: Record<string, ClusterSettingChange>;
  filter: ClusterSettingsFilter;
};

const initialState: ClusterSettingsState = {
  hostKey: '',
  settings: {},
  form: {},
  changes: {},
  filter: { name: '', showStatic: false },
};

export const clusterSettingsStore = createStore<ClusterSettingsState>(initialState);

export const clusterSettingsActions = {
  resetForHost(hostKey: string) {
    clusterSettingsStore.setState((state) => (state.hostKey === hostKey ? state : { ...initialState, hostKey }));
  },

  applyLoaded(hostKey: string, settings: Record<string, string>) {
    clusterSettingsStore.setState((state) => {
      if (state.hostKey !== hostKey) return state;
      const changed = new Set(Object.keys(state.changes));
      const form = changed.size
        ? Object.fromEntries(Object.entries(settings).map(([name, value]) => [name, changed.has(name) ? (state.form[name] ?? value) : value]))
        : settings;
      return { ...state, settings, form };
    });
  },

  updateSetting(name: string, value: string) {
    clusterSettingsStore.setState((state) => {
      const form = { ...state.form, [name]: value };
      const changes = { ...state.changes };
      if (value === state.settings[name]) {
        delete changes[name];
      } else {
        changes[name] = { transient: changes[name]?.transient ?? true, value };
      }
      return { ...state, form, changes };
    });
  },

  toggleTransient(name: string) {
    clusterSettingsStore.setState((state) => {
      const change = state.changes[name];
      if (!change) return state;
      return { ...state, changes: { ...state.changes, [name]: { ...change, transient: !change.transient } } };
    });
  },

  undoSetting(name: string) {
    clusterSettingsStore.setState((state) => {
      const form = { ...state.form, [name]: state.settings[name] ?? '' };
      const changes = { ...state.changes };
      delete changes[name];
      return { ...state, form, changes };
    });
  },

  setFilter(filter: Partial<ClusterSettingsFilter>) {
    clusterSettingsStore.setState((state) => ({ ...state, filter: { ...state.filter, ...filter } }));
  },

  clearChanges() {
    clusterSettingsStore.setState((state) => ({ ...state, changes: {} }));
  },
};
