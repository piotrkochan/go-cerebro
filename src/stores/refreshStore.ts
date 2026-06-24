import { createStore } from '@tanstack/react-store';

type RefreshState = {
  interval: number;
  tick: number;
};

const refreshIntervalKey = 'cerebro.refreshInterval';
const refreshIntervals = new Set([5000, 10000, 15000, 30000, 60000]);
const defaultRefreshInterval = 5000;

export const refreshStore = createStore<RefreshState>({ interval: savedRefreshInterval(), tick: 0 });

export const refreshActions = {
  setInterval(interval: number) {
    if (!refreshIntervals.has(interval)) return;
    window.localStorage.setItem(refreshIntervalKey, String(interval));
    refreshStore.setState((state) => ({ ...state, interval }));
  },
  tick() {
    refreshStore.setState((state) => ({ ...state, tick: state.tick + 1 }));
  },
};

function savedRefreshInterval() {
  const interval = Number(window.localStorage.getItem(refreshIntervalKey));
  return refreshIntervals.has(interval) ? interval : defaultRefreshInterval;
}
