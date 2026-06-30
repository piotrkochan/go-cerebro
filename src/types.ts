export type Alert = {
  id: number;
  kind: 'success' | 'danger' | 'info';
  text: string;
};

export type Notify = (kind: Alert['kind'], text: string) => void;
