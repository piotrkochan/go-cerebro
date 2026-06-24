export type TemplateFormValues = {
  body: string;
  name: string;
};

export const templateFormDefaults = (body: string): TemplateFormValues => ({
  body,
  name: '',
});
