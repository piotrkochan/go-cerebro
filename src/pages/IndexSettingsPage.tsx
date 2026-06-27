import { useEffect, useMemo, useState } from 'react';
import { Link } from '@tanstack/react-router';

import { indexSettingsGet, indexSettingsUpdate, type HostBodyWritable } from '../api/client';
import { Icon } from '../components/Icon';
import { SettingsPageLayout } from '../components/SettingsPageLayout';
import { SettingValueInput } from '../components/SettingValueInput';
import { isDynamicSetting, majorFromIndexVersionCreated, normalizeSettingValue, settingInput, settingSuggestions, type SettingInput } from '../settingsCatalog';
import type { Notify } from '../types';
import { errorMessage, textValue } from '../utils/format';

type Setting = { input: SettingInput; name: string; static: boolean };

const invalidIndexSettings = [
  'index.creation_date',
  'index.provided_name',
  'index.uuid',
  'index.version.created',
];

export function IndexSettingsPage({
  connection,
  index,
  majorVersion,
  notify,
}: {
  connection: HostBodyWritable;
  index: string;
  majorVersion?: number;
  notify: Notify;
}) {
  const [settings, setSettings] = useState<Record<string, string>>({});
  const [form, setForm] = useState<Record<string, string>>({});
  const [changes, setChanges] = useState<Record<string, string>>({});
  const [filter, setFilter] = useState({ name: '', showStatic: false });

  useEffect(() => {
    void load();
  }, [connection, index]);

  async function load() {
    if (!index) return;
    try {
      const result = await indexSettingsGet<true>({ body: { ...connection, index }, throwOnError: true });
      const flat = flattenIndexSettings(result.data.data, index, majorVersion);
      setSettings(flat);
      setForm(flat);
      setChanges({});
    } catch (error) {
      notify('danger', `Error loading index settings: ${errorMessage(error)}`);
    }
  }

  function updateSetting(name: string, value: string) {
    setForm((current) => ({ ...current, [name]: value }));
    setChanges((current) => {
      const next = { ...current };
      if (value === (settings[name] ?? '')) delete next[name];
      else next[name] = value;
      return next;
    });
  }

  async function save() {
    try {
      await indexSettingsUpdate<true>({ body: { ...connection, index, settings: changes }, throwOnError: true });
      notify('info', 'Settings successfully saved');
      await load();
    } catch (error) {
      notify('danger', `Error while saving settings: ${errorMessage(error)}`);
    }
  }

  const effectiveMajorVersion = majorVersion ?? majorFromIndexVersionCreated(settings['index.version.created']);
  const suggestions = useMemo(() => settingSuggestions('index', effectiveMajorVersion), [effectiveMajorVersion]);
  const groups = useMemo(
    () =>
      groupSettings(
        settingNames(form, suggestions)
          .filter(isValidIndexSetting)
          .map((name) => ({
            input: settingInput('index', name, effectiveMajorVersion),
            name,
            static: !isDynamicSetting('index', name, effectiveMajorVersion),
          })),
      ),
    [form, effectiveMajorVersion, suggestions],
  );
  const visibleGroups = groups
    .map((group) => ({
      ...group,
      settings: group.settings.filter((setting) => setting.name.includes(filter.name) && (!setting.static || filter.showStatic)),
    }))
    .filter((group) => group.settings.length > 0);
  const pending = Object.keys(changes).length;

  if (!index) {
    return <div className="text-center">missing index</div>;
  }

  return (
    <SettingsPageLayout
      actions={
        <Link className="btn btn-default whitespace-nowrap" search={{ host: connection.host }} to="/overview">
          <Icon name="undo" /> back to overview
        </Link>
      }
      filterName={filter.name}
      groupAriaLabel="Index settings groups"
      groupPrefix="index-setting"
      groups={visibleGroups}
      label="index settings"
      pendingCount={pending}
      pendingContent={Object.entries(changes).map(([setting, value]) => (
        <div className="px-3 py-2" key={setting}>
          <div className="break-all"><Icon name="cog" /> {setting}</div>
          <div className="mt-1 break-all text-[#d0d0d0]">
            <span className="info-text">updated to</span> {value || <span className="info-text">null</span>}
          </div>
          <div className="mt-2 text-right">
            <button className="btn btn-default btn-xs" title="undo" type="button" onClick={() => updateSetting(setting, settings[setting] ?? '')}>
              <Icon name="undo" />
            </button>
          </div>
        </div>
      ))}
      pendingFooter={<button className="btn btn-success" type="button" onClick={() => void save()}>save</button>}
      renderSetting={(setting) => (
        <div className="grid gap-2 px-3 py-2 lg:grid-cols-[minmax(260px,42%)_minmax(220px,1fr)] lg:items-center" key={setting.name}>
          <label className="mb-0 min-w-0 font-normal text-[#d0d0d0]">
            <span className="break-all">{setting.name}</span>{' '}
            {setting.static ? <Icon className="alert-warning" name="lock" /> : <span className="text-[#6f7579]">dynamic</span>}
          </label>
          <SettingValueInput
            disabled={setting.static}
            input={setting.input}
            value={form[setting.name] ?? ''}
            onChange={(value) => updateSetting(setting.name, value)}
          />
        </div>
      )}
      showStatic={filter.showStatic}
      title={index}
      onFilterNameChange={(name) => setFilter((value) => ({ ...value, name }))}
      onShowStaticChange={(showStatic) => setFilter((value) => ({ ...value, showStatic }))}
    />
  );
}

function flattenIndexSettings(raw: unknown, index: string, majorVersion?: number): Record<string, string> {
  const root = raw as Record<string, unknown>;
  const indexSettings = (root[index] ?? root) as { defaults?: unknown; settings?: unknown };
  const result: Record<string, string> = {};
  flatten(indexSettings.defaults, '', result, majorVersion);
  flatten(indexSettings.settings, '', result, majorVersion);
  return result;
}

function flatten(value: unknown, prefix: string, out: Record<string, string>, majorVersion?: number) {
  if (value && typeof value === 'object' && !Array.isArray(value)) {
    Object.entries(value as Record<string, unknown>).forEach(([key, nested]) => {
      flatten(nested, prefix ? `${prefix}.${key}` : key, out, majorVersion);
    });
    return;
  }
  if (prefix) out[prefix] = normalizeSettingValue('index', prefix, textValue(value), majorVersion);
}

function isValidIndexSetting(setting: string) {
  return invalidIndexSettings.every((invalid) => !setting.includes(invalid));
}

function settingNames(form: Record<string, string>, suggestions: string[]) {
  return Array.from(new Set([...Object.keys(form), ...suggestions])).filter((name) => !name.includes('<') && !name.includes('>'));
}

function groupSettings(settings: Setting[]) {
  const groups = new Map<string, Setting[]>();
  settings.forEach((setting) => {
    const group = indexSettingGroup(setting.name);
    groups.set(group, [...(groups.get(group) ?? []), setting]);
  });
  return Array.from(groups, ([name, groupSettings]) => ({
    name,
    settings: groupSettings.sort((a, b) => a.name.localeCompare(b.name)),
  })).sort((a, b) => a.name.localeCompare(b.name));
}

function indexSettingGroup(name: string) {
  if (name.startsWith('index.blocks.')) return 'blocks';
  if (name.startsWith('index.lifecycle.')) return 'lifecycle';
  if (name.startsWith('index.mapping.')) return 'mapping';
  if (name.startsWith('index.merge.')) return 'merge';
  if (name.startsWith('index.routing.allocation.')) return 'routing allocation';
  if (name.startsWith('index.search.slowlog.') || name.startsWith('index.indexing.slowlog.')) return 'slowlog';
  if (name.startsWith('index.translog.')) return 'translog';
  if (name.startsWith('index.default_pipeline') || name.startsWith('index.final_pipeline') || name.startsWith('index.search.default_pipeline')) return 'pipelines';
  if (name.startsWith('index.query.') || name.startsWith('index.queries.')) return 'query';
  if (name.startsWith('index.search.')) return 'search';
  if (name.startsWith('index.allocation.') || name.startsWith('index.routing.')) return 'routing';
  if (name.startsWith('index.max_')) return 'limits';
  if (name.startsWith('index.number_of_') || name === 'index.priority') return 'shards';
  if (name.startsWith('index.soft_deletes.')) return 'soft deletes';
  return 'general';
}
