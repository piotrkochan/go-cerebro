import { createStore } from '@tanstack/react-store';

import type { Alert } from '../types';

type AlertsState = {
  alerts: Alert[];
};

export const alertsStore = createStore<AlertsState>({ alerts: [] });

export const alertsActions = {
  clear() {
    alertsStore.setState(() => ({ alerts: [] }));
  },
  dismiss(id: number) {
    alertsStore.setState((state) => ({ alerts: state.alerts.filter((alert) => alert.id !== id) }));
  },
  notify(kind: Alert['kind'], text: string) {
    const id = Date.now();
    alertsStore.setState((state) => ({ alerts: [...state.alerts, { id, kind, text }] }));
    window.setTimeout(() => alertsActions.dismiss(id), 4500);
  },
};

export const notify = alertsActions.notify;
