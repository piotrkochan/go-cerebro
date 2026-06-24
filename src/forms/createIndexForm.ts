export type CreateIndexFormValues = {
  name: string;
  replicas: string;
  settings: string;
  shards: string;
  sourceIndex: string;
};

export const createIndexFormDefaults: CreateIndexFormValues = {
  name: '',
  replicas: '',
  settings: '',
  shards: '',
  sourceIndex: '',
};
