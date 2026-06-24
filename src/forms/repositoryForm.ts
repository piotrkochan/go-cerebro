export type RepositoryFormValues = {
  name: string;
  settings: string;
  type: string;
};

export const repositoryFormDefaults: RepositoryFormValues = {
  name: '',
  settings: '{}',
  type: 'fs',
};
