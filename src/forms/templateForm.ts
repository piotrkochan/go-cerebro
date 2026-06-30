export type TemplateFormValues = {
  body: string;
  kind: TemplateKind;
  name: string;
  wizard: TemplateWizardValues;
};

export type TemplateKind = 'index' | 'component' | 'legacy';

export type TemplateWizardValues = {
  aliases: string;
  componentTemplates: string[];
  dataStream: boolean;
  ignoreMissingComponentTemplates: string[];
  mappingFields: TemplateMappingField[];
  mappings: string;
  meta: string;
  order: string;
  pattern: string;
  priority: string;
  refreshInterval: string;
  replicas: string;
  routingShards: string;
  settings: string;
  shards: string;
  version: string;
};

export type TemplateMappingField = {
  analyzer: string;
  docValues: string;
  enabled: string;
  format: string;
  id: string;
  ignoreMalformed: string;
  index: string;
  name: string;
  parentId: string;
  store: string;
  type: string;
};

export const defaultTemplateWizard: TemplateWizardValues = {
  aliases: '{}',
  componentTemplates: [],
  dataStream: false,
  ignoreMissingComponentTemplates: [],
  mappingFields: [],
  mappings: '{}',
  meta: '{}',
  order: '',
  pattern: 'template pattern(e.g.: index_name_*)',
  priority: '',
  refreshInterval: '',
  replicas: '1',
  routingShards: '',
  settings: '{}',
  shards: '1',
  version: '',
};

export const templateFormDefaults = (body: string): TemplateFormValues => ({
  body,
  kind: 'index',
  name: '',
  wizard: defaultTemplateWizard,
});
