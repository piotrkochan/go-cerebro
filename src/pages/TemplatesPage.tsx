import { Link } from '@tanstack/react-router';
import { useEffect, useRef, useState, type ReactNode } from 'react';
import { useForm } from '@tanstack/react-form';

import { templatesCreate, templatesDelete, templatesGet, templatesList, type HostBodyWritable, type Template, type TemplateSummary } from '../api/client';
import { Button } from '../components/Button';
import { Checkbox } from '../components/Checkbox';
import { DataTable, SortHeader, type DataTableColumn } from '../components/DataTable';
import { Icon } from '../components/Icon';
import { Loading } from '../components/LegacyUi';
import { LazyJsonEditor } from '../components/LazyJsonEditor';
import { ConfirmModal } from '../components/Modal';
import { SplitPane } from '../components/SplitPane';
import { defaultTemplateWizard, templateFormDefaults, type TemplateFormValues, type TemplateKind, type TemplateMappingField, type TemplateWizardValues } from '../forms/templateForm';
import type { Notify } from '../types';
import { clusterPath } from '../utils/connection';
import { errorMessage, formatJson, parseJson, textValue } from '../utils/format';
import { nextSort, sortByText, type SortState } from '../utils/sort';

type TemplateSortKey = 'data_stream' | 'name' | 'pattern';
type TemplateEditorMode = 'wizard' | 'json';
type TemplateWithKind = Template & { kind?: TemplateKind };
type TemplateListItem = TemplateSummary & { kind?: TemplateKind };

const mappingTypes = [
  'keyword',
  'text',
  'long',
  'integer',
  'short',
  'byte',
  'double',
  'float',
  'scaled_float',
  'boolean',
  'date',
  'ip',
  'object',
  'nested',
  'flattened',
  'geo_point',
  'geo_shape',
  'dense_vector',
  'rank_feature',
  'search_as_you_type',
];

const templateBaseObject = {
  index_patterns: ['template pattern(e.g.: index_name_*)'],
  priority: 100,
  template: {
    aliases: {},
    mappings: {},
    settings: {},
  },
};

const componentTemplateBaseObject = {
  template: {
    aliases: {},
    mappings: {},
    settings: {},
  },
};

const legacyTemplateBaseObject = {
  aliases: {},
  mappings: {},
  settings: {},
  template: 'template pattern(e.g.: index_name_*)',
};

function defaultTemplateKind(majorVersion?: number): TemplateKind {
  return (majorVersion ?? 9) < 7 ? 'legacy' : 'index';
}

function normalizeTemplateKind(kind: TemplateKind | undefined, majorVersion?: number): TemplateKind {
  if (kind) return kind;
  if ((majorVersion ?? 9) < 7) return 'legacy';
  return kind ?? defaultTemplateKind(majorVersion);
}

function templateBase(kind: TemplateKind) {
  switch (kind) {
  case 'component':
    return formatJson(componentTemplateBaseObject);
  case 'legacy':
    return formatJson(legacyTemplateBaseObject);
  default:
    return formatJson(templateBaseObject);
  }
}

function templateDetailKey(connection: HostBodyWritable, kind: TemplateKind, name: string) {
  return `${connection.host}:${kind}:${name}`;
}

function templateTargetOptions(connection: HostBodyWritable) {
  return {
    path: clusterPath(connection),
  };
}

export function TemplatesPage({
  connection,
  majorVersion,
  notify,
  onKindChange,
  refreshTick,
  selectedKind,
  selectedTemplate,
}: {
  connection: HostBodyWritable;
  majorVersion?: number;
  notify: Notify;
  onKindChange?: (kind: TemplateKind) => void;
  refreshTick: number;
  selectedKind?: TemplateKind;
  selectedTemplate?: string;
}) {
  const [templates, setTemplates] = useState<TemplateListItem[]>([]);
  const initialKind = normalizeTemplateKind(selectedKind, majorVersion);
  const [activeKind, setActiveKind] = useState<TemplateKind>(initialKind);
  const [filter, setFilter] = useState({ name: '', pattern: '' });
  const [deleteTemplate, setDeleteTemplate] = useState<TemplateListItem | null>(null);
  const [editorMode, setEditorMode] = useState<TemplateEditorMode>('wizard');
  const [loadingTemplates, setLoadingTemplates] = useState(true);
  const [loadingTemplateDetail, setLoadingTemplateDetail] = useState(Boolean(selectedTemplate));
  const [templateDetailError, setTemplateDetailError] = useState<{ key: string; message: string } | null>(null);
  const [sort, setSort] = useState<SortState<TemplateSortKey>>({ key: 'name', order: 'asc' });
  const openedTemplate = useRef('');
  const requestedTemplate = useRef('');
  const defaultBody = templateBase(initialKind);
  const form = useForm({
    defaultValues: { ...templateFormDefaults(defaultBody), kind: initialKind, wizard: wizardFromTemplateBody(defaultBody, initialKind) },
    onSubmit: async ({ value }) => {
      await save(value);
    },
  });

  useEffect(() => {
    void loadTemplates();
  }, [connection, refreshTick]);

  useEffect(() => {
    const nextKind = normalizeTemplateKind(selectedKind, majorVersion);
    if (!selectedKind) {
      onKindChange?.(nextKind);
    }
    if (activeKind !== nextKind && !selectedTemplate) {
      setActiveKind(nextKind);
      resetForm(nextKind);
    }
    if (activeKind !== nextKind && selectedTemplate) {
      setActiveKind(nextKind);
    }
    if (!selectedTemplate && openedTemplate.current) {
      openedTemplate.current = '';
      requestedTemplate.current = '';
      setTemplateDetailError(null);
      resetForm(nextKind);
    }
    if (selectedTemplate) {
      const key = templateDetailKey(connection, nextKind, selectedTemplate);
      if (openedTemplate.current !== key && requestedTemplate.current !== key) {
        void loadTemplateDetail(nextKind, selectedTemplate);
      }
    }
  }, [connection, majorVersion, selectedKind, selectedTemplate]);

  async function loadTemplates() {
    setLoadingTemplates(true);
    try {
      const result = await templatesList<true>({ ...templateTargetOptions(connection), throwOnError: true });
      const items = ((result.data.items ?? []) as TemplateListItem[]).sort((left, right) => left.name.localeCompare(right.name));
      setTemplates(items);
    } catch (error) {
      notify('danger', `Error while loading templates: ${errorMessage(error)}`);
    } finally {
      setLoadingTemplates(false);
    }
  }

  async function loadTemplateDetail(kind: TemplateKind, name: string) {
    const key = templateDetailKey(connection, kind, name);
    requestedTemplate.current = key;
    setLoadingTemplateDetail(true);
    setTemplateDetailError(null);
    try {
      const result = await templatesGet<true>({
        path: { ...clusterPath(connection), kind, name },
        throwOnError: true,
      });
      edit(result.data as TemplateWithKind);
      openedTemplate.current = key;
      setTemplateDetailError(null);
    } catch (error) {
      const message = errorMessage(error) || 'request failed before a readable error response was returned';
      setTemplateDetailError({ key, message });
      notify('danger', `Error loading template: ${message}`);
    } finally {
      if (requestedTemplate.current === key) {
        requestedTemplate.current = '';
      }
      setLoadingTemplateDetail(false);
    }
  }

  async function save(values: TemplateFormValues) {
    const name = values.name.trim();
    const body = parseJson(values.body);
    if (!name) {
      notify('danger', 'template name is required');
      return;
    }
    if (!body || typeof body !== 'object' || Array.isArray(body)) {
      notify('danger', 'template body must be a JSON object');
      return;
    }
    try {
      await templatesCreate<true>({
        body,
        path: { ...clusterPath(connection), kind: values.kind, name },
        throwOnError: true,
      });
      notify('info', templates.some((template) => template.name === name && templateKind(template) === values.kind) ? 'Template successfully updated' : 'Template successfully created');
      await loadTemplates();
    } catch (error) {
      notify('danger', `Error creating template: ${errorMessage(error)}`);
    }
  }

  async function remove(templateName: string, kind: TemplateKind) {
    try {
      await templatesDelete<true>({
        path: { ...clusterPath(connection), kind, name: templateName },
        throwOnError: true,
      });
      notify('info', 'Template successfully deleted');
      await loadTemplates();
    } catch (error) {
      notify('danger', `Error deleting template: ${errorMessage(error)}`);
    }
  }

  function edit(template: TemplateWithKind) {
    const kind = templateKind(template);
    setActiveKind(kind);
    form.setFieldValue('kind', kind);
    form.setFieldValue('name', template.name);
    const body = formatJson(template.template);
    form.setFieldValue('body', body);
    form.setFieldValue('wizard', wizardFromTemplateBody(body, kind));
    setEditorMode('wizard');
  }

  function resetForm(kind = activeKind) {
    openedTemplate.current = '';
    requestedTemplate.current = '';
    form.setFieldValue('name', '');
    form.setFieldValue('kind', kind);
    const body = templateBase(kind);
    form.setFieldValue('body', body);
    form.setFieldValue('wizard', wizardFromTemplateBody(body, kind));
    setEditorMode('wizard');
  }

  function updateWizard(current: TemplateWizardValues, next: Partial<TemplateWizardValues>) {
    const value = { ...current, ...next };
    form.setFieldValue('wizard', value);
    form.setFieldValue('body', formatJson(templateBodyFromWizard(value, activeKind)));
  }

  function switchMode(mode: TemplateEditorMode, body: string, kind: TemplateKind) {
    if (mode === 'wizard') {
      form.setFieldValue('wizard', wizardFromTemplateBody(body, kind));
    }
    setEditorMode(mode);
  }

  function changeKind(kind: TemplateKind) {
    setActiveKind(kind);
    resetForm(kind);
    onKindChange?.(kind);
  }

  const filtered = sortByText(
    templates.filter((template) => {
      const pattern = templatePattern(template);
      const patternMatches = activeKind === 'component' || pattern.includes(filter.pattern);
      return templateKind(template) === activeKind && template.name.includes(filter.name) && patternMatches;
    }),
    sort,
    templateSortValue,
  );
  const splitManagedTemplates = activeKind === 'index' || activeKind === 'component';
  const notManagedTemplates = splitManagedTemplates ? filtered.filter((template) => !isManagedTemplate(template)) : filtered;
  const managedTemplates = splitManagedTemplates ? filtered.filter(isManagedTemplate) : [];
  const componentTemplateNames = templates.filter((template) => templateKind(template) === 'component').map((template) => template.name).sort();
  const supportsComposableTemplates = (majorVersion ?? 9) >= 7;
  const selectedTemplateKind = normalizeTemplateKind(selectedKind, majorVersion);
  const selectedTemplateKey = selectedTemplate ? templateDetailKey(connection, selectedTemplateKind, selectedTemplate) : '';
  const waitingForSelectedTemplate = Boolean(selectedTemplate && openedTemplate.current !== selectedTemplateKey && templateDetailError?.key !== selectedTemplateKey);
  const selectedTemplateMissing = Boolean(selectedTemplate && !loadingTemplateDetail && templateDetailError?.key === selectedTemplateKey);
  const kindCounts = {
    component: templates.filter((template) => templateKind(template) === 'component').length,
    index: templates.filter((template) => templateKind(template) === 'index').length,
    legacy: templates.filter((template) => templateKind(template) === 'legacy').length,
  };
  const templateColumns: DataTableColumn<TemplateListItem>[] = [
    {
      header: <SortHeader name="name" sort={sort} onSort={(name) => setSort((value) => nextSort(value, name))}>name</SortHeader>,
      key: 'name',
      render: (template) => <><Icon name={templateKind(template) === 'component' ? 'puzzle' : 'book'} /> {template.name}</>,
    },
    ...(activeKind === 'component' ? [] : [{
      header: <SortHeader name="pattern" sort={sort} onSort={(name) => setSort((value) => nextSort(value, name))}>pattern</SortHeader>,
      key: 'pattern',
      render: (template: TemplateListItem) => templatePattern(template),
    }]),
    ...(activeKind === 'index' ? [{
      header: <SortHeader name="data_stream" sort={sort} onSort={(name) => setSort((value) => nextSort(value, name))}>data stream</SortHeader>,
      key: 'data_stream',
      render: (template: TemplateListItem) => template.data_stream ? <span className="label bg-[#4b5f6b] text-[#eceeef]">yes</span> : <span className="info-text">-</span>,
    }] : []),
    {
      className: 'text-right',
      header: 'actions',
      headerClassName: 'text-right',
      key: 'actions',
      render: (template) => (
        <span className="inline-flex items-center justify-end gap-[10px]">
          <Link
            aria-label={`edit template ${template.name}`}
            className="btn btn-default btn-xs"
            search={(previous) => ({ ...previous, kind: templateKind(template), template: template.name })}
            title="edit template"
            to="/templates"
          >
            <Icon name="pencil" />
          </Link>
          <button aria-label={`delete template ${template.name}`} className="btn btn-danger btn-xs" title="delete template" type="button" onClick={() => setDeleteTemplate(template)}>
            <Icon name="trash" />
          </button>
        </span>
      ),
    },
  ];

  return (
    <>
      {deleteTemplate ? (
        <ConfirmModal
          body={
            <>
              Delete template <strong>{deleteTemplate.name}</strong>? This operation cannot be undone.
            </>
          }
          confirmLabel={
            <>
              <Icon name="trash" /> delete template
            </>
          }
          onClose={() => setDeleteTemplate(null)}
          onConfirm={() => remove(deleteTemplate.name, templateKind(deleteTemplate))}
          title="delete template"
        />
      ) : null}
      <div className="mb-[15px] border-b border-[#55595c]">
        <KindTabs
          activeKind={activeKind}
          counts={kindCounts}
          supportsComposableTemplates={supportsComposableTemplates}
          onChange={changeKind}
        />
      </div>
      <SplitPane
        storageKey="cerebro.templatesSplitPercent"
        left={
          <>
            <h4>
              existing templates <small className="info-text">({filtered.length})</small>
            </h4>
            <div className="row">
              <div className="col-md-9">
                {templates.length ? (
                  <div className="row">
                    <div className="col-md-6 form-group">
                      <input className="form-control" placeholder="template name" value={filter.name} onChange={(event) => setFilter((value) => ({ ...value, name: event.target.value }))} />
                    </div>
                    {activeKind === 'component' ? null : (
                      <div className="col-md-6 form-group">
                        <input className="form-control" placeholder="template pattern" value={filter.pattern} onChange={(event) => setFilter((value) => ({ ...value, pattern: event.target.value }))} />
                      </div>
                    )}
                  </div>
                ) : null}
              </div>
              <div className="col-md-3 form-group text-right">
                <Link className="btn btn-success btn-xs" search={(previous) => ({ ...previous, kind: activeKind, template: undefined })} to="/templates" onClick={() => resetForm()}>
                  <Icon name="plus" /> new template
                </Link>
              </div>
              <div className="col-xs-12">
                {loadingTemplates && templates.length === 0 ? (
                  <Loading label="loading templates" />
                ) : splitManagedTemplates ? (
                  <>
                    <h4>my {templateKindLabel(activeKind).toLowerCase()} <small className="info-text">({notManagedTemplates.length})</small></h4>
                    <DataTable columns={templateColumns} getRowKey={(template) => template.name} rows={notManagedTemplates} />

                    <h4 className="mt-[20px]">managed {templateKindLabel(activeKind).toLowerCase()} <small className="info-text">({managedTemplates.length})</small></h4>
                    <DataTable columns={templateColumns} getRowKey={(template) => template.name} rows={managedTemplates} />
                  </>
                ) : (
                  <DataTable columns={templateColumns} getRowKey={(template) => template.name} rows={filtered} />
                )}
              </div>
            </div>
          </>
        }
        right={
          waitingForSelectedTemplate ? (
            <TemplateEditorState>
              <Loading label={`loading template ${selectedTemplate}`} />
            </TemplateEditorState>
          ) : selectedTemplateMissing ? (
            <TemplateEditorState>
              <h4>template not found</h4>
              <p className="info-text">
                No {templateKindSingular(selectedTemplateKind)} named <strong>{selectedTemplate}</strong> was returned by Elasticsearch.
              </p>
              <p className="info-text">{templateDetailError?.message}</p>
              <Button icon="refresh" size="xs" variant="default" onClick={() => loadTemplateDetail(selectedTemplateKind, selectedTemplate!)}>
                retry
              </Button>
            </TemplateEditorState>
          ) : (
          <>
            <form.Subscribe selector={(state) => ({ kind: state.values.kind, name: state.values.name })}>
              {({ kind, name }) => (
                <div className="mb-[12px] flex flex-wrap items-center justify-between gap-[10px]">
                  <h4 className="!m-0">{templates.some((template) => template.name === name && templateKind(template) === kind) ? `update ${templateKindSingular(kind)} ${name}` : `create ${templateKindSingular(kind)}`}</h4>
                  <span className="label bg-[#434749] text-[#eceeef]">{templateKindLabel(kind)}</span>
                </div>
              )}
            </form.Subscribe>
            <div className="row">
              <div className="col-xs-12">
                <div className="form-group">
                  <form.Field name="name">
                    {(field) => <input className="form-control" placeholder="name" value={field.state.value} onChange={(event) => field.handleChange(event.target.value)} />}
                  </form.Field>
                </div>
              </div>
              <div className="col-xs-12">
                <div className="form-group">
                  <form.Field name="body">
                    {(field) => (
                      <>
                        <div className="mb-[15px] inline-flex overflow-hidden border border-[#55595c]">
                          <button
                            className={`px-[12px] py-[6px] ${editorMode === 'wizard' ? 'bg-[#434749] text-white' : 'bg-transparent text-[#eceeef]'}`}
                            type="button"
                            onClick={() => switchMode('wizard', field.state.value, activeKind)}
                          >
                            wizard
                          </button>
                          <button
                            className={`border-l border-[#55595c] px-[12px] py-[6px] ${editorMode === 'json' ? 'bg-[#434749] text-white' : 'bg-transparent text-[#eceeef]'}`}
                            type="button"
                            onClick={() => switchMode('json', field.state.value, activeKind)}
                          >
                            json
                          </button>
                        </div>
                        {editorMode === 'wizard' ? (
                          <form.Subscribe selector={(state) => state.values.wizard}>
                            {(wizard) => (
                              <TemplateWizard
                                componentTemplates={componentTemplateNames}
                                kind={activeKind}
                                majorVersion={majorVersion}
                                value={wizard}
                                onChange={(next) => updateWizard(wizard, next)}
                              />
                            )}
                          </form.Subscribe>
                        ) : (
                          <LazyJsonEditor height={600} value={field.state.value} onChange={field.handleChange} />
                        )}
                      </>
                    )}
                  </form.Field>
                </div>
              </div>
              <div className="col-xs-12 text-right">
                <form.Subscribe selector={(state) => state.values.name}>
                  {(name) => {
                    const editMode = templates.some((template) => template.name === name && templateKind(template) === activeKind);
                    return (
                      <Button icon={editMode ? 'save' : 'plus'} type="submit" variant={editMode ? 'warning' : 'success'} onClick={() => void form.handleSubmit()}>
                        {editMode ? 'update' : 'create'}
                      </Button>
                    );
                  }}
                </form.Subscribe>
              </div>
            </div>
          </>
          )
        }
      />
    </>
  );
}

function TemplateEditorState({ children }: { children: ReactNode }) {
  return (
    <div className="border border-[#55595c] bg-[#343739] p-[20px]">
      {children}
    </div>
  );
}

function KindTabs({
  activeKind,
  counts,
  onChange,
  supportsComposableTemplates,
}: {
  activeKind: TemplateKind;
  counts: Record<TemplateKind, number>;
  onChange: (kind: TemplateKind) => void;
  supportsComposableTemplates: boolean;
}) {
  const kinds: TemplateKind[] = supportsComposableTemplates ? ['index', 'component', 'legacy'] : ['legacy'];
  return (
    <div className="flex flex-wrap items-end gap-[4px]">
      {kinds.map((kind) => {
        const active = activeKind === kind;
        return (
        <button
          className={`relative -mb-px flex min-w-[150px] items-center justify-between gap-[10px] border border-[#55595c] px-[12px] py-[7px] text-left transition-colors ${active ? 'border-b-[#383c3e] bg-[#383c3e] text-white' : 'bg-[#2f3234] text-[#cfd3d6] hover:bg-[#3a3e40]'}`}
          key={kind}
          type="button"
          onClick={() => onChange(kind)}
        >
          <span className="inline-flex min-w-0 items-center gap-[6px]">
            <Icon name={kind === 'component' ? 'puzzle' : 'book'} />
            <span className="truncate">{templateKindLabel(kind)}</span>
          </span>
          <span className={`label ${active ? 'bg-[#1f2123]' : 'bg-[#55595c]'} text-[#eceeef]`}>{counts[kind]}</span>
        </button>
        );
      })}
    </div>
  );
}

function TemplateWizard({
  componentTemplates,
  kind,
  majorVersion,
  onChange,
  value,
}: {
  componentTemplates: string[];
  kind: TemplateKind;
  majorVersion?: number;
  onChange: (next: Partial<TemplateWizardValues>) => void;
  value: TemplateWizardValues;
}) {
  const availableMappingTypes = mappingTypesForVersion(majorVersion);
  const [selectedFieldId, setSelectedFieldId] = useState('');
  const selectedField = value.mappingFields.find((field) => field.id === selectedFieldId) ?? value.mappingFields[0] ?? null;
  function addField(parentId = '') {
    const field = newMappingField(parentId);
    onChange({ mappingFields: [...value.mappingFields, field] });
    setSelectedFieldId(field.id);
  }

  function updateField(id: string, next: Partial<TemplateMappingField>) {
    const normalized = next.type && !availableMappingTypes.includes(next.type) ? { ...next, type: availableMappingTypes[0] } : next;
    onChange({
      mappingFields: value.mappingFields.map((field) => field.id === id ? { ...field, ...normalized } : field),
    });
  }

  function removeField(id: string) {
    const removeIds = descendantFieldIds(value.mappingFields, id);
    onChange({ mappingFields: value.mappingFields.filter((field) => !removeIds.has(field.id)) });
    if (removeIds.has(selectedFieldId)) {
      setSelectedFieldId('');
    }
  }

  return (
    <div className="space-y-[15px]">
      {kind === 'index' || kind === 'legacy' ? (
        <section className="border border-[#55595c] bg-[#343739] p-[12px]">
          <SectionTitle subtitle={kind === 'legacy' ? 'legacy template match and order' : 'patterns, priority and data stream mode'} title={kind === 'legacy' ? 'match' : 'logistics'} />
          <div className="row">
            <WizardInput label="index patterns" placeholder="logs-*, metrics-*" value={value.pattern} onChange={(pattern) => onChange({ pattern })} />
            {kind === 'legacy' ? (
              <WizardInput label="order" placeholder="0" value={value.order} onChange={(order) => onChange({ order })} />
            ) : (
              <WizardInput label="priority" placeholder="100" value={value.priority} onChange={(priority) => onChange({ priority })} />
            )}
            <WizardInput label="version" placeholder="1" value={value.version} onChange={(version) => onChange({ version })} />
            {kind === 'index' ? (
              <div className="col-sm-6 form-group">
                <Checkbox checked={value.dataStream} label="data stream template" onChange={(dataStream) => onChange({ dataStream })} />
              </div>
            ) : null}
          </div>
        </section>
      ) : null}

      {kind === 'index' ? (
        <section className="border border-[#55595c] bg-[#343739] p-[12px]">
          <SectionTitle subtitle="reusable template parts applied before local settings" title="component templates" />
          <div className="row">
            <div className="col-sm-6 form-group">
              <ComponentTemplatePicker
                label="composed of"
                options={componentTemplates}
                value={value.componentTemplates}
                onChange={(componentTemplates) => onChange({ componentTemplates })}
              />
            </div>
            <div className="col-sm-6 form-group">
              <ComponentTemplatePicker
                label="ignore missing"
                options={componentTemplates}
                value={value.ignoreMissingComponentTemplates}
                onChange={(ignoreMissingComponentTemplates) => onChange({ ignoreMissingComponentTemplates })}
              />
            </div>
          </div>
        </section>
      ) : null}

      <section className="border border-[#55595c] bg-[#343739] p-[12px]">
        <SectionTitle subtitle="common index-level settings plus raw JSON for advanced options" title="index settings" />
        <div className="row">
          <WizardInput label="number of shards" placeholder="1" value={value.shards} onChange={(shards) => onChange({ shards })} />
          <WizardInput label="number of replicas" placeholder="1" value={value.replicas} onChange={(replicas) => onChange({ replicas })} />
          <WizardInput label="refresh interval" placeholder="1s" value={value.refreshInterval} onChange={(refreshInterval) => onChange({ refreshInterval })} />
          <WizardInput label="routing shards" placeholder="optional" value={value.routingShards} onChange={(routingShards) => onChange({ routingShards })} />
        </div>
        <label className="form-label">additional settings JSON</label>
        <LazyJsonEditor height={170} value={value.settings} onChange={(settings) => onChange({ settings })} />
      </section>

      <section className="border border-[#55595c] bg-[#343739] p-[12px]">
        <div className="mb-[10px] flex items-center justify-between">
          <div>
            <h4 className="!m-0 !text-[14px]">mappings</h4>
            <div className="info-text">field types are limited to Elasticsearch {majorVersion ? `${majorVersion}.x` : 'cluster'} support</div>
          </div>
          <button className="btn btn-success btn-xs" type="button" onClick={() => addField()}>
            <Icon name="plus" /> add field
          </button>
        </div>
        <div className="grid gap-[15px] lg:grid-cols-[minmax(260px,38%)_minmax(320px,1fr)]">
          <MappingFieldsTree
            fields={value.mappingFields}
            selectedId={selectedField?.id ?? ''}
            onAddChild={addField}
            onRemove={removeField}
            onSelect={setSelectedFieldId}
          />
          <MappingFieldEditor
            availableTypes={availableMappingTypes}
            field={selectedField}
            fields={value.mappingFields}
            onAddChild={addField}
            onChange={(next) => selectedField ? updateField(selectedField.id, next) : undefined}
            onRemove={(id) => removeField(id)}
          />
        </div>
        <details className="mt-[12px]">
          <summary className="normal-action info-text">additional mappings JSON</summary>
          <LazyJsonEditor height={180} value={value.mappings} onChange={(mappings) => onChange({ mappings })} />
        </details>
      </section>

      <section className="border border-[#55595c] bg-[#343739] p-[12px]">
        <SectionTitle subtitle="optional aliases included in the generated template" title="aliases" />
        <LazyJsonEditor height={160} value={value.aliases} onChange={(aliases) => onChange({ aliases })} />
      </section>

      {kind === 'index' || kind === 'component' ? (
        <section className="border border-[#55595c] bg-[#343739] p-[12px]">
          <SectionTitle subtitle="stored in _meta for operators and tooling" title="metadata" />
          <LazyJsonEditor height={130} value={value.meta} onChange={(meta) => onChange({ meta })} />
        </section>
      ) : null}
    </div>
  );
}

function SectionTitle({ subtitle, title }: { subtitle: string; title: string }) {
  return (
    <div className="mb-[10px]">
      <h4 className="!m-0 !text-[14px]">{title}</h4>
      <div className="info-text">{subtitle}</div>
    </div>
  );
}

function ComponentTemplatePicker({
  label,
  onChange,
  options,
  value,
}: {
  label: string;
  onChange: (value: string[]) => void;
  options: string[];
  value: string[];
}) {
  const [selected, setSelected] = useState('');
  const available = options.filter((option) => !value.includes(option));

  useEffect(() => {
    if (selected && !available.includes(selected)) {
      setSelected('');
    }
  }, [available.join('\0'), selected]);

  function addSelected() {
    if (!selected || value.includes(selected)) return;
    onChange([...value, selected]);
    setSelected('');
  }

  return (
    <>
      <label className="form-label">{label}</label>
      <div className="flex gap-[8px]">
        <select className="form-control min-w-0 flex-1" value={selected} onChange={(event) => setSelected(event.target.value)}>
          <option value="">{available.length ? 'select component template' : 'no component templates available'}</option>
          {available.map((option) => <option key={option}>{option}</option>)}
        </select>
        <Button disabled={!selected} icon="plus" size="xs" variant="default" onClick={addSelected}>
          add
        </Button>
      </div>
      <div className="mt-[8px] flex min-h-[31px] flex-wrap gap-[6px]">
        {value.length ? value.map((item) => (
          <span className="inline-flex max-w-full items-center gap-[6px] border border-[#55595c] bg-[#2f3234] px-[8px] py-[4px]" key={item}>
            <span className="truncate">{item}</span>
            <button
              aria-label={`remove ${item}`}
              className="normal-action shrink-0 text-[#eceeef]"
              type="button"
              onClick={() => onChange(value.filter((candidate) => candidate !== item))}
            >
              x
            </button>
          </span>
        )) : <span className="info-text py-[4px]">none selected</span>}
      </div>
    </>
  );
}

function WizardInput({
  label,
  onChange,
  placeholder,
  value,
}: {
  label: string;
  onChange: (value: string) => void;
  placeholder: string;
  value: string;
}) {
  return (
    <div className="col-sm-6 form-group">
      <label className="form-label">{label}</label>
      <input className="form-control font-mono" placeholder={placeholder} value={value} onChange={(event) => onChange(event.target.value)} />
    </div>
  );
}

function MappingFieldsTree({
  fields,
  onAddChild,
  onRemove,
  onSelect,
  selectedId,
}: {
  fields: TemplateMappingField[];
  onAddChild: (parentId?: string) => void;
  onRemove: (id: string) => void;
  onSelect: (id: string) => void;
  selectedId: string;
}) {
  const orderedFields = orderedMappingFields(fields);
  return (
    <div className="border border-[#55595c] bg-[#343739]">
      <div className="border-b border-[#55595c] px-[10px] py-[7px] font-bold">fields</div>
      {orderedFields.length ? (
        <div className="divide-y divide-[#434749]">
          {orderedFields.map((field) => {
            const depth = fieldDepth(fields, field);
            const selected = selectedId === field.id;
            return (
              <button
                className={`flex w-full items-center gap-[8px] px-[10px] py-[7px] text-left hover:bg-[#434749] ${selected ? 'bg-[#434749]' : ''}`}
                key={field.id}
                type="button"
                onClick={() => onSelect(field.id)}
              >
                <span className={`flex min-w-0 flex-1 items-center gap-[6px] ${fieldIndentClass(depth)}`}>
                  {depth > 0 ? <span className="inline-block h-px w-[18px] shrink-0 bg-[#8b8f95]" /> : null}
                  <span className="min-w-0 truncate">{field.name || '(unnamed)'}</span>
                </span>
                <span className="label bg-[#55595c] text-[#eceeef]">{field.type}</span>
                {isContainerField(field) ? (
                  <span
                    className="btn btn-default btn-xs"
                    title="add child field"
                    onClick={(event) => {
                      event.stopPropagation();
                      onAddChild(field.id);
                    }}
                  >
                    <Icon name="plus" />
                  </span>
                ) : null}
                <span
                  className="btn btn-danger btn-xs"
                  title="remove field"
                  onClick={(event) => {
                    event.stopPropagation();
                    onRemove(field.id);
                  }}
                >
                  <Icon name="trash" />
                </span>
              </button>
            );
          })}
        </div>
      ) : (
        <div className="info-text px-[10px] py-[12px]">no fields defined</div>
      )}
    </div>
  );
}

function MappingFieldEditor({
  availableTypes,
  field,
  fields,
  onAddChild,
  onChange,
  onRemove,
}: {
  availableTypes: string[];
  field: TemplateMappingField | null;
  fields: TemplateMappingField[];
  onAddChild: (parentId?: string) => void;
  onChange: (next: Partial<TemplateMappingField>) => void;
  onRemove: (id: string) => void;
}) {
  if (!field) {
    return (
      <div className="border border-[#55595c] p-[12px]">
        <div className="info-text">Select a field or add a new one.</div>
      </div>
    );
  }
  return (
    <div className="border border-[#55595c] p-[12px]">
      <div className="mb-[10px] flex items-center justify-between gap-[10px]">
        <h4 className="!m-0 break-all">{fieldPath(fields, field)}</h4>
        <span className="inline-flex items-center gap-[10px]">
          {isContainerField(field) ? (
            <button className="btn btn-default btn-xs" type="button" onClick={() => onAddChild(field.id)}>
              <Icon name="plus" /> child
            </button>
          ) : null}
          <button className="btn btn-danger btn-xs" type="button" onClick={() => onRemove(field.id)}>
            <Icon name="trash" />
          </button>
        </span>
      </div>
      <div className="row">
        <WizardInput label="field name" placeholder="user.name" value={field.name} onChange={(name) => onChange({ name })} />
        <div className="col-sm-6 form-group">
          <label className="form-label">parent</label>
          <select className="form-control" value={field.parentId} onChange={(event) => onChange({ parentId: event.target.value })}>
            <option value="">root</option>
            {fields
              .filter((candidate) => candidate.id !== field.id && isContainerField(candidate))
              .map((candidate) => (
                <option key={candidate.id} value={candidate.id}>
                  {fieldPath(fields, candidate)}
                </option>
              ))}
          </select>
        </div>
        <div className="col-sm-6 form-group">
          <label className="form-label">type</label>
          <select className="form-control" value={field.type} onChange={(event) => onChange({ type: event.target.value })}>
            {availableTypes.map((type) => <option key={type}>{type}</option>)}
          </select>
        </div>
      </div>
      <h4>options</h4>
      <MappingFieldOptions field={field} onChange={onChange} />
    </div>
  );
}

function MappingFieldOptions({
  field,
  onChange,
}: {
  field: TemplateMappingField;
  onChange: (next: Partial<TemplateMappingField>) => void;
}) {
  const common = (
    <div className="grid grid-cols-2 gap-[6px]">
      <DefaultBooleanSelect label="index" value={field.index} onChange={(index) => onChange({ index })} />
      <DefaultBooleanSelect label="doc_values" value={field.docValues} onChange={(docValues) => onChange({ docValues })} />
      <DefaultBooleanSelect label="store" value={field.store} onChange={(store) => onChange({ store })} />
      <DefaultBooleanSelect label="ignore_malformed" value={field.ignoreMalformed} onChange={(ignoreMalformed) => onChange({ ignoreMalformed })} />
    </div>
  );
  if (field.type === 'object' || field.type === 'nested') {
    return (
      <div className="space-y-[6px]">
        <DefaultBooleanSelect label="enabled" value={field.enabled} onChange={(enabled) => onChange({ enabled })} />
        {common}
      </div>
    );
  }
  if (field.type === 'text') {
    return (
      <div className="space-y-[6px]">
        <input className="form-control" placeholder="analyzer" value={field.analyzer} onChange={(event) => onChange({ analyzer: event.target.value })} />
        {common}
      </div>
    );
  }
  if (field.type === 'date') {
    return (
      <div className="space-y-[6px]">
        <input className="form-control" placeholder="format, e.g. strict_date_optional_time||epoch_millis" value={field.format} onChange={(event) => onChange({ format: event.target.value })} />
        {common}
      </div>
    );
  }
  if (field.type === 'scaled_float') {
    return (
      <div className="space-y-[6px]">
        <input className="form-control" placeholder="scaling_factor" value={field.format} onChange={(event) => onChange({ format: event.target.value })} />
        {common}
      </div>
    );
  }
  return common;
}

function DefaultBooleanSelect({
  label,
  onChange,
  value,
}: {
  label: string;
  onChange: (value: string) => void;
  value: string;
}) {
  return (
    <label className="m-0 grid grid-cols-[90px_minmax(0,1fr)] items-center gap-[6px] font-normal">
      <span className="info-text">{label}</span>
      <select className="form-control" value={value} onChange={(event) => onChange(event.target.value)}>
        <option value="">default</option>
        <option value="true">true</option>
        <option value="false">false</option>
      </select>
    </label>
  );
}

function templateBodyFromWizard(value: TemplateWizardValues, kind: TemplateKind) {
  const settings = objectValue(parseJson(value.settings));
  setNumberString(settings, 'number_of_shards', value.shards);
  setNumberString(settings, 'number_of_replicas', value.replicas);
  setString(settings, 'refresh_interval', value.refreshInterval);
  setNumberString(settings, 'number_of_routing_shards', value.routingShards);

  const template: Record<string, unknown> = {
    aliases: objectValue(parseJson(value.aliases)),
    mappings: mergeMappings(objectValue(parseJson(value.mappings)), value.mappingFields),
    settings,
  };
  if (kind === 'legacy') {
    const body: Record<string, unknown> = { ...template };
    body.template = value.pattern.trim();
    setNumberString(body, 'order', value.order);
    setNumberString(body, 'version', value.version);
    return body;
  }
  const body: Record<string, unknown> = { template };
  const meta = objectValue(parseJson(value.meta));
  if (Object.keys(meta).length) body._meta = meta;
  setNumberString(body, 'version', value.version);
  if (kind === 'component') {
    return body;
  }
  body.index_patterns = templatePatternList(value.pattern);
  if (value.componentTemplates.length) body.composed_of = value.componentTemplates;
  if (value.ignoreMissingComponentTemplates.length) body.ignore_missing_component_templates = value.ignoreMissingComponentTemplates;
  if (value.dataStream) body.data_stream = {};
  setNumberString(body, 'priority', value.priority);
  return body;
}

function wizardFromTemplateBody(body: string, kind: TemplateKind): TemplateWizardValues {
  const root = objectValue(parseJson(body));
  const definition = kind === 'legacy' ? root : objectValue(root.template);
  const settings = objectValue(definition.settings);
  const mappings = objectValue(definition.mappings);
  const extracted = fieldsFromMappings(mappings);
  const wizard = {
    ...defaultTemplateWizard,
    aliases: formatJson(objectValue(definition.aliases)),
    componentTemplates: stringArrayField(root.composed_of),
    dataStream: root.data_stream !== undefined,
    ignoreMissingComponentTemplates: stringArrayField(root.ignore_missing_component_templates),
    mappingFields: extracted.fields,
    mappings: formatJson(extracted.rest),
    meta: formatJson(objectValue(root._meta)),
    order: stringField(root.order, defaultTemplateWizard.order),
    pattern: patternField(root, defaultTemplateWizard.pattern),
    priority: stringField(root.priority, defaultTemplateWizard.priority),
    refreshInterval: settingField(settings, 'refresh_interval', defaultTemplateWizard.refreshInterval),
    replicas: settingField(settings, 'number_of_replicas', defaultTemplateWizard.replicas),
    routingShards: settingField(settings, 'number_of_routing_shards', defaultTemplateWizard.routingShards),
    settings: formatJson(stripKnownTemplateSettings(settings)),
    shards: settingField(settings, 'number_of_shards', defaultTemplateWizard.shards),
    version: stringField(root.version, defaultTemplateWizard.version),
  };
  return wizard;
}

function mergeMappings(base: Record<string, unknown>, fields: TemplateMappingField[]) {
  const mappings = { ...base };
  const generatedProperties = mappingPropertiesFromFields(fields);
  if (Object.keys(generatedProperties).length) {
    mappings.properties = mergeProperties(objectValue(mappings.properties), generatedProperties);
  }
  return mappings;
}

function mappingPropertiesFromFields(fields: TemplateMappingField[]) {
  const roots: Record<string, unknown> = {};
  const byParent = new Map<string, TemplateMappingField[]>();
  fields.forEach((field) => {
    const key = field.parentId || '';
    byParent.set(key, [...(byParent.get(key) ?? []), field]);
  });
  fillProperties(roots, '', byParent);
  return roots;
}

function fillProperties(target: Record<string, unknown>, parentId: string, byParent: Map<string, TemplateMappingField[]>) {
  for (const field of byParent.get(parentId) ?? []) {
    const name = field.name.trim();
    if (!name) continue;
    const body: Record<string, unknown> = { type: field.type };
    if (field.type === 'text') setString(body, 'analyzer', field.analyzer);
    if (field.type === 'date') setString(body, 'format', field.format);
    if (field.type === 'scaled_float') setNumberString(body, 'scaling_factor', field.format);
    if ((field.type === 'object' || field.type === 'nested') && field.enabled) {
      body.enabled = field.enabled === 'true';
    }
    setBooleanString(body, 'index', field.index);
    setBooleanString(body, 'doc_values', field.docValues);
    setBooleanString(body, 'store', field.store);
    setBooleanString(body, 'ignore_malformed', field.ignoreMalformed);
    const children: Record<string, unknown> = {};
    fillProperties(children, field.id, byParent);
    if (Object.keys(children).length) {
      body.properties = children;
    }
    target[name] = body;
  }
}

function fieldsFromMappings(mappings: Record<string, unknown>) {
  const rest = { ...mappings };
  delete rest.properties;
  return { fields: fieldsFromProperties(objectValue(mappings.properties), ''), rest };
}

function fieldsFromProperties(properties: Record<string, unknown>, parentId: string): TemplateMappingField[] {
  return Object.entries(properties).flatMap(([name, value]) => {
    const body = objectValue(value);
    const id = newFieldId();
    const field: TemplateMappingField = {
      analyzer: stringField(body.analyzer, ''),
      docValues: booleanField(body.doc_values),
      enabled: typeof body.enabled === 'boolean' ? String(body.enabled) : '',
      format: stringField(body.format ?? body.scaling_factor, ''),
      id,
      ignoreMalformed: booleanField(body.ignore_malformed),
      index: booleanField(body.index),
      name,
      parentId,
      store: booleanField(body.store),
      type: stringField(body.type, 'object'),
    };
    return [field, ...fieldsFromProperties(objectValue(body.properties), id)];
  });
}

function mergeProperties(base: Record<string, unknown>, generated: Record<string, unknown>) {
  return { ...base, ...generated };
}

function newMappingField(parentId = ''): TemplateMappingField {
  return {
    analyzer: '',
    docValues: '',
    enabled: '',
    format: '',
    id: newFieldId(),
    ignoreMalformed: '',
    index: '',
    name: '',
    parentId,
    store: '',
    type: 'keyword',
  };
}

function newFieldId() {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function descendantFieldIds(fields: TemplateMappingField[], id: string) {
  const ids = new Set([id]);
  let changed = true;
  while (changed) {
    changed = false;
    for (const field of fields) {
      if (!ids.has(field.id) && ids.has(field.parentId)) {
        ids.add(field.id);
        changed = true;
      }
    }
  }
  return ids;
}

function isContainerField(field: TemplateMappingField) {
  return field.type === 'object' || field.type === 'nested';
}

function fieldPath(fields: TemplateMappingField[], field: TemplateMappingField): string {
  const parent = fields.find((candidate) => candidate.id === field.parentId);
  return parent ? `${fieldPath(fields, parent)}.${field.name || '(unnamed)'}` : field.name || '(unnamed)';
}

function orderedMappingFields(fields: TemplateMappingField[]) {
  const ordered: TemplateMappingField[] = [];
  const append = (parentId: string) => {
    fields.filter((field) => field.parentId === parentId).forEach((field) => {
      ordered.push(field);
      append(field.id);
    });
  };
  append('');
  fields.filter((field) => field.parentId && !fields.some((candidate) => candidate.id === field.parentId)).forEach((field) => ordered.push(field));
  return ordered;
}

function fieldDepth(fields: TemplateMappingField[], field: TemplateMappingField): number {
  if (!field.parentId) return 0;
  const parent = fields.find((candidate) => candidate.id === field.parentId);
  return parent ? fieldDepth(fields, parent) + 1 : 0;
}

function fieldIndentClass(depth: number) {
  const classes = ['pl-0', 'pl-4', 'pl-8', 'pl-12', 'pl-16', 'pl-20'];
  return classes[Math.min(depth, classes.length - 1)];
}

function mappingTypesForVersion(majorVersion?: number) {
  const version = majorVersion ?? 9;
  return mappingTypes.filter((type) => {
    if (version < 7 && ['flattened', 'rank_feature', 'search_as_you_type', 'dense_vector'].includes(type)) return false;
    if (version < 6 && ['alias'].includes(type)) return false;
    return true;
  });
}

function stripKnownTemplateSettings(settings: Record<string, unknown>) {
  const next = { ...settings };
  delete next.number_of_shards;
  delete next.number_of_replicas;
  delete next.refresh_interval;
  delete next.number_of_routing_shards;
  return next;
}

function settingField(settings: Record<string, unknown>, key: string, fallback: string) {
  return stringField(settings[key], fallback);
}

function objectValue(value: unknown): Record<string, unknown> {
  return value && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : {};
}

function stringField(value: unknown, fallback: string) {
  const text = textValue(value);
  return text || fallback;
}

function setString(target: Record<string, unknown>, key: string, value: string) {
  const trimmed = value.trim();
  if (trimmed) {
    target[key] = trimmed;
  }
}

function setNumberString(target: Record<string, unknown>, key: string, value: string) {
  const trimmed = value.trim();
  if (!trimmed) return;
  const parsed = Number(trimmed);
  target[key] = Number.isFinite(parsed) ? parsed : trimmed;
}

function setBooleanString(target: Record<string, unknown>, key: string, value: string) {
  if (value === 'true') target[key] = true;
  if (value === 'false') target[key] = false;
}

function booleanField(value: unknown) {
  return typeof value === 'boolean' ? String(value) : '';
}

function stringArrayField(value: unknown) {
  return Array.isArray(value) ? value.map((item) => textValue(item)).filter(Boolean) : [];
}

function templateKind(template: TemplateListItem | TemplateWithKind): TemplateKind {
  if (template.kind === 'index' || template.kind === 'component' || template.kind === 'legacy') {
    return template.kind;
  }
  const body = 'template' in template ? template.template as Record<string, unknown> : {};
  if (Array.isArray(body.index_patterns) || Array.isArray(body.composed_of) || body.data_stream !== undefined) return 'index';
  if (body.template && typeof body.template === 'object') return 'component';
  return 'legacy';
}

function isManagedTemplate(template: TemplateListItem | TemplateWithKind) {
  if ('managed' in template) {
    return template.managed;
  }
  const body = objectValue(template.template);
  const meta = objectValue(body._meta);
  const managed = meta.managed ?? meta.system ?? body.managed;
  const managedBy = textValue(meta.managed_by ?? meta.managedBy ?? body.managed_by);
  return managed === true || managed === 'true' || Boolean(managedBy) || template.name.startsWith('.');
}

function templateKindLabel(kind: TemplateKind) {
  switch (kind) {
  case 'component':
    return 'component templates';
  case 'legacy':
    return 'legacy templates';
  default:
    return 'index templates';
  }
}

function templateKindSingular(kind: TemplateKind) {
  switch (kind) {
  case 'component':
    return 'component template';
  case 'legacy':
    return 'legacy template';
  default:
    return 'index template';
  }
}

function templatePattern(template: TemplateListItem | TemplateWithKind) {
  const kind = templateKind(template);
  if (kind === 'component') return 'reusable component';
  if ('pattern' in template) return template.pattern ?? '';
  if ('template' in template) return patternField(template.template as Record<string, unknown>, '');
  return '';
}

function patternField(root: Record<string, unknown>, fallback: string) {
  const indexPatterns = root.index_patterns;
  if (Array.isArray(indexPatterns)) {
    const pattern = indexPatterns.map((value) => textValue(value)).filter(Boolean).join(', ');
    if (pattern) return pattern;
  }
  return stringField(root.template, fallback);
}

function templatePatternList(pattern: string) {
  const patterns = pattern.split(',').map((value) => value.trim()).filter(Boolean);
  return patterns.length ? patterns : [defaultTemplateWizard.pattern];
}

function templateSortValue(template: TemplateListItem, key: TemplateSortKey) {
  if (key === 'name') return template.name;
  if (key === 'data_stream') return template.data_stream ? '1' : '0';
  return templatePattern(template);
}
