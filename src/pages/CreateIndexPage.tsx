import { useEffect, useMemo, useState } from 'react';
import { useForm } from '@tanstack/react-form';
import { useDebouncedValue } from '@tanstack/react-pacer';

import {
  createIndexCreate,
  createIndexGetMetadata,
  overview,
  templatesList,
  type HostBodyWritable,
  type TemplateSummary,
} from '../api/client';
import { Icon } from '../components/Icon';
import { LazyJsonEditor } from '../components/LazyJsonEditor';
import { SplitPane } from '../components/SplitPane';
import { createIndexFormDefaults, type CreateIndexFormValues } from '../forms/createIndexForm';
import type { Notify } from '../types';
import { clusterPath } from '../utils/connection';
import { errorMessage, formatJson } from '../utils/format';

export function CreateIndexPage({
  connection,
  majorVersion,
  notify,
  refreshTick,
}: {
  connection: HostBodyWritable;
  majorVersion?: number;
  notify: Notify;
  refreshTick: number;
}) {
  const [indices, setIndices] = useState<string[]>([]);
  const [templates, setTemplates] = useState<TemplateSummary[]>([]);
  const [indexName, setIndexName] = useState(createIndexFormDefaults.name);
  const [debouncedIndexName, nameDebouncer] = useDebouncedValue(
    indexName.trim(),
    { wait: 2000 },
    (state) => ({ isPending: state.isPending }),
  );
  const form = useForm({
    defaultValues: createIndexFormDefaults,
    onSubmit: async ({ value }) => {
      await createIndex(value);
    },
  });
  const matchingTemplates = useMemo(
    () => matchingIndexTemplates(templates, debouncedIndexName),
    [debouncedIndexName, templates],
  );

  useEffect(() => {
    let ignore = false;

    async function load() {
      try {
        const [indicesResult, templatesResult] = await Promise.all([
          overview<true>({ path: clusterPath(connection), throwOnError: true }),
          templatesList<true>({ path: clusterPath(connection), throwOnError: true }),
        ]);
        if (!ignore) {
          setIndices(
            (indicesResult.data.indices ?? [])
              .filter((index) => !index.data_stream)
              .map((index) => index.name)
              .sort((left, right) => left.localeCompare(right)),
          );
          setTemplates((templatesResult.data.items ?? []).sort(templateSort));
        }
      } catch (error) {
        notify('danger', errorMessage(error));
      }
    }

    void load();
    return () => {
      ignore = true;
    };
  }, [connection, notify, refreshTick]);

  async function loadIndexMetadata(index: string) {
    form.setFieldValue('sourceIndex', index);
    if (!index) return;

    try {
      const result = await createIndexGetMetadata<true>({
        path: { ...clusterPath(connection), index },
        throwOnError: true,
      });
      const body = {
        aliases: {},
        mappings: result.data.mappings ?? {},
        settings: result.data.settings ?? {},
      };
      sanitizeCreateIndexBody(body, majorVersion);
      const currentShards = form.getFieldValue('shards').trim();
      const currentReplicas = form.getFieldValue('replicas').trim();

      if (currentShards) {
        setIndexSetting(body, 'number_of_shards', currentShards);
      } else {
        const loadedShards = readIndexSetting(body, 'number_of_shards');
        if (loadedShards !== '') form.setFieldValue('shards', loadedShards);
      }

      if (currentReplicas) {
        setIndexSetting(body, 'number_of_replicas', currentReplicas);
      } else {
        const loadedReplicas = readIndexSetting(body, 'number_of_replicas');
        if (loadedReplicas !== '') form.setFieldValue('replicas', loadedReplicas);
      }

      form.setFieldValue('settings', formatJson(body));
    } catch (error) {
      notify('danger', `Error while loading index settings: ${errorMessage(error)}`);
    }
  }

  function updateIndexSetting(key: 'number_of_replicas' | 'number_of_shards', value: string) {
    const body = parseJsonObject(form.getFieldValue('settings'));
    if (!body) return;
    setIndexSetting(body, key, value);
    form.setFieldValue('settings', formatJson(body));
  }

  async function createIndex(values: CreateIndexFormValues) {
    if (!values.name.trim()) {
      notify('danger', 'You must specify a valid index name');
      return;
    }
    const dataStreamTemplates = matchingIndexTemplates(templates, values.name.trim()).filter((template) => template.data_stream);
    if (dataStreamTemplates.length) {
      notify('danger', `Index name matches data stream template ${dataStreamTemplates.map((template) => template.name).join(', ')}. Create a data stream instead.`);
      return;
    }

    let metadata: Record<string, unknown>;
    if (values.settings.trim()) {
      let parsed: unknown;
      try {
        parsed = JSON.parse(values.settings);
      } catch (error) {
        notify('danger', `Malformed settings: ${errorMessage(error)}`);
        return;
      }
      if (!isRecord(parsed)) {
        notify('danger', 'Malformed settings: root value must be a JSON object');
        return;
      }
      metadata = parsed;
      applyIndexSettingOverride(metadata, 'number_of_shards', values.shards);
      applyIndexSettingOverride(metadata, 'number_of_replicas', values.replicas);
    } else {
      metadata = { aliases: {}, mappings: {}, settings: { index: {} } };
      applyIndexSettingOverride(metadata, 'number_of_shards', values.shards);
      applyIndexSettingOverride(metadata, 'number_of_replicas', values.replicas);
    }

    try {
      await createIndexCreate<true>({
        body: metadata,
        path: { ...clusterPath(connection), index: values.name.trim() },
        throwOnError: true,
      });
      notify('success', 'Index successfully created');
    } catch (error) {
      notify('danger', `Error while creating index: ${errorMessage(error)}`);
    }
  }

  return (
    <SplitPane
      storageKey="cerebro.createIndexSplitPercent"
      left={
        <>
          <div className="row">
            <div className="col-xs-12">
              <div className="form-group">
                <label className="form-label">name</label>
                <form.Field name="name">
                  {(field) => (
                    <input
                      className="form-control"
                      placeholder="index name"
                      type="text"
                      value={field.state.value}
                      onChange={(event) => {
                        field.handleChange(event.target.value);
                        setIndexName(event.target.value);
                      }}
                    />
                  )}
                </form.Field>
                <MatchingTemplates
                  indexName={indexName}
                  pending={nameDebouncer.state.isPending && indexName.trim() !== debouncedIndexName}
                  templates={matchingTemplates}
                />
              </div>
            </div>
          </div>
          <div className="row">
            <div className="col-sm-6">
              <div className="form-group">
                <label className="form-label">number of shards</label>
                <form.Field name="shards">
                  {(field) => (
                    <input
                      className="form-control"
                      placeholder="# of shards"
                      type="number"
                      value={field.state.value}
                      onChange={(event) => {
                        field.handleChange(event.target.value);
                        updateIndexSetting('number_of_shards', event.target.value);
                      }}
                    />
                  )}
                </form.Field>
              </div>
            </div>
            <div className="col-sm-6">
              <div className="form-group">
                <label className="form-label">number of replicas</label>
                <form.Field name="replicas">
                  {(field) => (
                    <input
                      className="form-control"
                      placeholder="# of replicas"
                      type="number"
                      value={field.state.value}
                      onChange={(event) => {
                        field.handleChange(event.target.value);
                        updateIndexSetting('number_of_replicas', event.target.value);
                      }}
                    />
                  )}
                </form.Field>
              </div>
            </div>
          </div>
          <div className="row">
            <div className="col-xs-12">
              <div className="form-group">
                <label className="form-label">load settings from existing index</label>
                <form.Field name="sourceIndex">
                  {(field) => (
                    <select
                      className="form-control"
                      value={field.state.value}
                      onChange={(event) => void loadIndexMetadata(event.target.value)}
                    >
                      <option value="" />
                      {indices.map((index) => (
                        <option key={index}>{index}</option>
                      ))}
                    </select>
                  )}
                </form.Field>
              </div>
            </div>
          </div>
          <div className="row">
            <div className="col-lg-12">
              <span className="pull-right">
                <button className="btn btn-primary" type="button" onClick={() => void form.handleSubmit()}>
                  <Icon name="check" /> Create
                </button>
              </span>
            </div>
          </div>
        </>
      }
      right={
        <div className="form-group">
          <label className="form-label">settings</label>
          <form.Field name="settings">
            {(field) => <LazyJsonEditor height={600} value={field.state.value} onChange={field.handleChange} />}
          </form.Field>
        </div>
      }
    />
  );
}

function MatchingTemplates({
  indexName,
  pending,
  templates,
}: {
  indexName: string;
  pending: boolean;
  templates: TemplateSummary[];
}) {
  if (!indexName.trim()) return null;

  return (
    <div className="mt-[6px] border border-[#55595c] bg-[#303335] px-2 py-[6px] text-[12px]">
      <div className="mb-[4px] flex items-center justify-between gap-2 text-[#dfe3e6]">
        <span>matching templates</span>
        {pending ? <span className="info-text">checking...</span> : null}
      </div>
      {templates.length ? (
        <div className="flex flex-wrap gap-[5px]">
          {templates.map((template) => (
            <span key={`${template.kind}:${template.name}`} className="inline-flex items-center gap-[5px] border border-[#55595c] bg-[#373a3c] px-[6px] py-[2px]">
              <span className="text-[#8bdbff]">{template.kind}</span>
              <span>{template.name}</span>
              {template.data_stream ? <span className="label label-warning">data stream</span> : null}
              {template.pattern ? <span className="info-text">{template.pattern}</span> : null}
            </span>
          ))}
        </div>
      ) : (
        <span className="info-text">none</span>
      )}
    </div>
  );
}

function matchingIndexTemplates(templates: TemplateSummary[], indexName: string) {
  const name = indexName.trim();
  if (!name) return [];

  return templates.filter((template) => {
    if (template.kind === 'component') return false;
    return templatePatterns(template).some((pattern) => globMatch(pattern, name));
  });
}

function templatePatterns(template: TemplateSummary) {
  return (template.pattern ?? '')
    .split(',')
    .map((pattern) => pattern.trim())
    .filter(Boolean);
}

function globMatch(pattern: string, value: string) {
  const regex = new RegExp(`^${pattern.replace(/[.+^${}()|[\]\\]/g, '\\$&').replace(/\*/g, '.*').replace(/\?/g, '.')}$`);
  return regex.test(value);
}

function templateSort(left: TemplateSummary, right: TemplateSummary) {
  return `${left.kind}:${left.name}`.localeCompare(`${right.kind}:${right.name}`);
}

function parseJsonObject(value: string) {
  try {
    const parsed: unknown = JSON.parse(value || '{}');
    return isRecord(parsed) ? parsed : null;
  } catch {
    return null;
  }
}

function readIndexSetting(body: Record<string, unknown>, key: 'number_of_replicas' | 'number_of_shards') {
  const settings = isRecord(body.settings) ? body.settings : {};
  const indexSettings = isRecord(settings.index) ? settings.index : settings;
  const value = indexSettings[key];
  return value === undefined || value === null ? '' : String(value);
}

function setIndexSetting(body: Record<string, unknown>, key: 'number_of_replicas' | 'number_of_shards', rawValue: string) {
  if (!isRecord(body.settings)) body.settings = {};
  const settings = body.settings as Record<string, unknown>;
  const indexSettings = isRecord(settings.index) ? settings.index : settings;
  const value = rawValue.trim();

  if (!value) {
    delete indexSettings[key];
    return;
  }

  indexSettings[key] = /^\d+$/.test(value) ? Number(value) : value;
}

function applyIndexSettingOverride(body: Record<string, unknown>, key: 'number_of_replicas' | 'number_of_shards', rawValue: string) {
  if (!rawValue.trim()) return;
  setIndexSetting(body, key, rawValue);
}

function sanitizeCreateIndexBody(body: Record<string, unknown>, majorVersion?: number) {
  sanitizeCreateIndexSettings(body);
  sanitizeCreateIndexMappings(body, majorVersion);
}

function sanitizeCreateIndexSettings(body: Record<string, unknown>) {
  if (!isRecord(body.settings)) return;
  const indexSettings = isRecord(body.settings.index) ? body.settings.index : body.settings;

  delete indexSettings.creation_date;
  delete indexSettings.provided_name;
  delete indexSettings.uuid;
  delete indexSettings.version;
}

function sanitizeCreateIndexMappings(body: Record<string, unknown>, majorVersion?: number) {
  if ((majorVersion ?? 9) < 7 || !isRecord(body.mappings)) return;

  const mappingTypes = Object.keys(body.mappings).filter((key) => key !== '_meta' && key !== '_source' && key !== 'dynamic_templates');
  if (mappingTypes.length !== 1) return;

  const typedMapping = body.mappings[mappingTypes[0]];
  if (!isRecord(typedMapping)) return;

  body.mappings = typedMapping;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
