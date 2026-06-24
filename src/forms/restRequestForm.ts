export type RestRequestFormValues = {
  body: string;
  method: string;
  path: string;
};

export const restRequestFormDefaults: RestRequestFormValues = {
  body: '',
  method: 'POST',
  path: '',
};
