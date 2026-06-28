export type RepositoryFormValues = {
  name: string;
  settings: string;
  type: string;
};

export const repositoryFormDefaults: RepositoryFormValues = {
  name: '',
  settings: '{\n  "location": "",\n  "compress": true\n}',
  type: 'fs',
};
