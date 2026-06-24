import type { HostBodyWritable } from './api/client';

export type Alert = {
  id: number;
  kind: 'success' | 'danger' | 'info';
  text: string;
};

export type Notify = (kind: Alert['kind'], text: string) => void;

export type ConnectionAuth = Pick<HostBodyWritable, 'username' | 'password'>;
