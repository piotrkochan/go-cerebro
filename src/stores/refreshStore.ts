import { createStore } from '@tanstack/react-store';

type RefreshState = {
  interval: number;
  tick: number;
};

export const refreshStore = createStore<RefreshState>({ interval: 5000, tick: 0 });

export const refreshActions = {
  setInterval(interval: number) {
    refreshStore.setState((state) => ({ ...state, interval }));
  },
  tick() {
    refreshStore.setState((state) => ({ ...state, tick: state.tick + 1 }));
  },
};
