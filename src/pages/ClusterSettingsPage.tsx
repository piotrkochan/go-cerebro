import { useEffect, useMemo } from 'react';
import { Link } from '@tanstack/react-router';
import { useStore } from '@tanstack/react-store';

import { clusterSettingsGet, clusterSettingsSave, type HostBodyWritable } from '../api/client';
import { Button } from '../components/Button';
import { Icon } from '../components/Icon';
import { SettingsPageLayout } from '../components/SettingsPageLayout';
import { SettingValueInput } from '../components/SettingValueInput';
import { isDynamicSetting, normalizeSettingValue, settingInput, settingSuggestions, type SettingInput } from '../settingsCatalog';
import { clusterSettingsActions, clusterSettingsStore } from '../stores/clusterSettingsStore';
import type { Notify } from '../types';
import { clusterPath } from '../utils/connection';
import { errorMessage, textValue } from '../utils/format';

type Setting = { input: SettingInput; name: string; static: boolean };

export function ClusterSettingsPage({
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
  const { changes, filter, form, settings } = useStore(clusterSettingsStore);
  const hostKey = connection.host;

  useEffect(() => {
    clusterSettingsActions.resetForHost(hostKey);
    void load();
  }, [connection, hostKey, refreshTick]);

  async function load() {
    try {
      const result = await clusterSettingsGet<true>({ path: clusterPath(connection), throwOnError: true });
      const flat = flattenSettings(result.data.data, majorVersion);
      clusterSettingsActions.applyLoaded(hostKey, flat);
    } catch (error) {
      notify('danger', `Error loading cluster settings: ${errorMessage(error)}`);
    }
  }

  async function save() {
    const body = { persistent: {} as Record<string, string | null>, transient: {} as Record<string, string | null> };
    Object.entries(changes).forEach(([setting, change]) => {
      body[change.transient ? 'transient' : 'persistent'][setting] = change.value.length > 0 ? change.value : null;
    });
    try {
      await clusterSettingsSave<true>({ body, path: clusterPath(connection), throwOnError: true });
      notify('info', 'Settings successfully saved');
      clusterSettingsActions.clearChanges();
      await load();
    } catch (error) {
      notify('danger', `Error while saving settings: ${errorMessage(error)}`);
    }
  }

  const suggestions = useMemo(() => settingSuggestions('cluster', majorVersion), [majorVersion]);
  const groups = useMemo(
    () =>
      groupSettings(
        settingNames(form, suggestions).map((name) => ({
          input: settingInput('cluster', name, majorVersion),
          name,
          static: !isDynamicSetting('cluster', name, majorVersion),
        })),
      ),
    [form, majorVersion, suggestions],
  );
  const visibleGroups = groups
    .map((group) => ({
      ...group,
      settings: group.settings.filter((setting) => setting.name.includes(filter.name) && (!setting.static || filter.showStatic)),
    }))
    .filter((group) => group.settings.length > 0);
  const pending = Object.keys(changes).length;

  return (
    <SettingsPageLayout
      actions={
        <Link className="btn btn-default whitespace-nowrap" search={{ host: connection.host }} to="/overview">
          <Icon name="undo" /> back to overview
        </Link>
      }
      filterName={filter.name}
      groupAriaLabel="Cluster settings groups"
      groupPrefix="cluster-setting"
      groups={visibleGroups}
      label="cluster settings"
      pendingCount={pending}
      pendingContent={Object.entries(changes).map(([setting, change]) => (
        <div className="px-3 py-2" key={setting}>
          <div className="break-all"><Icon name="cog" /> {setting}</div>
          <div className="mt-1 break-all text-[#d0d0d0]">
            <span className="info-text">updated to</span> {change.value || <span className="info-text">null</span>}
          </div>
          <div className="mt-2 flex items-center justify-between">
            <button className="btn btn-default btn-xs" type="button" onClick={() => clusterSettingsActions.toggleTransient(setting)}>
              {change.transient ? 'transient' : 'persistent'}
            </button>
            <button className="btn btn-default btn-xs" title="undo" type="button" onClick={() => clusterSettingsActions.undoSetting(setting)}>
              <Icon name="undo" />
            </button>
          </div>
        </div>
      ))}
      pendingFooter={<Button icon="save" variant="success" onClick={() => void save()}>save</Button>}
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
            onChange={(value) => clusterSettingsActions.updateSetting(setting.name, value)}
          />
        </div>
      )}
      showStatic={filter.showStatic}
      title={connection.host}
      onFilterNameChange={(name) => clusterSettingsActions.setFilter({ name })}
      onShowStaticChange={(showStatic) => clusterSettingsActions.setFilter({ showStatic })}
    />
  );
}

function settingNames(form: Record<string, string>, suggestions: string[]) {
  return Array.from(new Set([...Object.keys(form), ...suggestions])).filter((name) => !name.includes('<') && !name.includes('>'));
}

function flattenSettings(raw: unknown, majorVersion?: number): Record<string, string> {
  const result: Record<string, string> = {};
  const root = raw as { defaults?: unknown; persistent?: unknown; transient?: unknown };
  ['defaults', 'persistent', 'transient'].forEach((section) => flatten(root?.[section as keyof typeof root], '', result, majorVersion));
  return result;
}

function flatten(value: unknown, prefix: string, out: Record<string, string>, majorVersion?: number) {
  if (value && typeof value === 'object' && !Array.isArray(value)) {
    Object.entries(value as Record<string, unknown>).forEach(([key, nested]) => {
      flatten(nested, prefix ? `${prefix}.${key}` : key, out, majorVersion);
    });
    return;
  }
  if (prefix) out[prefix] = normalizeSettingValue('cluster', prefix, textValue(value), majorVersion);
}

function groupSettings(settings: Setting[]) {
  const groups = new Map<string, Setting[]>();
  settings.forEach((setting) => {
    const group = clusterSettingGroup(setting.name);
    groups.set(group, [...(groups.get(group) ?? []), setting]);
  });
  return Array.from(groups, ([name, groupSettings]) => ({ name, settings: groupSettings.sort((a, b) => a.name.localeCompare(b.name)) })).sort((a, b) => a.name.localeCompare(b.name));
}

function clusterSettingGroup(name: string) {
  if (name.startsWith('action.')) return 'actions';
  if (name.startsWith('cluster.blocks.')) return 'blocks';
  if (name.startsWith('cluster.info.')) return 'cluster info';
  if (name.startsWith('cluster.max_')) return 'limits';
  if (name.startsWith('cluster.persistent_tasks.')) return 'persistent tasks';
  if (name.startsWith('cluster.remote.')) return 'remote clusters';
  if (name.startsWith('cluster.routing.allocation.')) return 'allocation';
  if (name.startsWith('cluster.routing.rebalance.')) return 'rebalance';
  if (name.startsWith('cluster.service.')) return 'cluster service';
  if (name.startsWith('discovery.')) return 'discovery';
  if (name.startsWith('gateway.')) return 'gateway';
  if (name.startsWith('indices.breaker.') || name.startsWith('network.breaker.')) return 'breakers';
  if (name.startsWith('indices.lifecycle.')) return 'lifecycle';
  if (name.startsWith('indices.mapping.')) return 'mapping';
  if (name.startsWith('indices.recovery.')) return 'recovery';
  if (name.startsWith('indices.store.') || name.startsWith('indices.ttl.') || name.startsWith('indices.id_field_data.')) return 'indices';
  if (name.startsWith('ingest.')) return 'ingest';
  if (name.startsWith('script.')) return 'scripts';
  if (name.startsWith('search.')) return 'search';
  if (name.startsWith('transport.')) return 'transport';
  if (name.startsWith('xpack.ml.')) return 'machine learning';
  if (name.startsWith('xpack.monitoring.')) return 'monitoring';
  if (name.startsWith('xpack.security.')) return 'security';
  if (name.startsWith('xpack.watcher.')) return 'watcher';
  return 'general';
}
