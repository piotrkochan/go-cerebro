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
  settings: `{
  "settings": {
    "index": {
      "number_of_shards": 1,
      "number_of_replicas": 0
    }
  },
  "mappings": {},
  "aliases": {}
}`,
  shards: '',
  sourceIndex: '',
};
