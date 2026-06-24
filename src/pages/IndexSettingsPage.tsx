import { useEffect, useMemo, useState } from 'react';

import { indexSettingsGet, indexSettingsUpdate, type HostBodyWritable } from '../api/client';
import { Icon } from '../components/Icon';
import { SplitPane } from '../components/SplitPane';
import type { Notify } from '../types';
import { errorMessage, textValue } from '../utils/format';

type Setting = { name: string; static: boolean };

const dynamicIndexSettings = new Set([
  'index.allocation.max_retries',
  'index.auto_expand_replicas',
  'index.blocks.metadata',
  'index.blocks.read',
  'index.blocks.read_only',
  'index.blocks.read_only_allow_delete',
  'index.blocks.write',
  'index.gc_deletes',
  'index.indexing.slowlog.level',
  'index.indexing.slowlog.reformat',
  'index.indexing.slowlog.source',
  'index.indexing.slowlog.threshold.index.debug',
  'index.indexing.slowlog.threshold.index.info',
  'index.indexing.slowlog.threshold.index.trace',
  'index.indexing.slowlog.threshold.index.warn',
  'index.mapping.depth.limit',
  'index.mapping.nested_fields.limit',
  'index.mapping.total_fields.limit',
  'index.max_adjacency_matrix_filters',
  'index.max_refresh_listeners',
  'index.max_rescore_window',
  'index.max_result_window',
  'index.max_slices_per_scroll',
  'index.merge.policy.expunge_deletes_allowed',
  'index.merge.policy.floor_segment',
  'index.merge.policy.max_merge_at_once',
  'index.merge.policy.max_merge_at_once_explicit',
  'index.merge.policy.max_merged_segment',
  'index.merge.policy.reclaim_deletes_weight',
  'index.merge.policy.segments_per_tier',
  'index.merge.scheduler.auto_throttle',
  'index.merge.scheduler.max_merge_count',
  'index.merge.scheduler.max_thread_count',
  'index.number_of_replicas',
  'index.priority',
  'index.refresh_interval',
  'index.requests.cache.enable',
  'index.routing.allocation.enable',
  'index.routing.allocation.total_shards_per_node',
  'index.routing.rebalance.enable',
  'index.search.slowlog.level',
  'index.search.slowlog.threshold.fetch.debug',
  'index.search.slowlog.threshold.fetch.info',
  'index.search.slowlog.threshold.fetch.trace',
  'index.search.slowlog.threshold.fetch.warn',
  'index.search.slowlog.threshold.query.debug',
  'index.search.slowlog.threshold.query.info',
  'index.search.slowlog.threshold.query.trace',
  'index.search.slowlog.threshold.query.warn',
  'index.translog.durability',
  'index.translog.flush_threshold_size',
  'index.unassigned.node_left.delayed_timeout',
  'index.write.wait_for_active_shards',
]);

const invalidIndexSettings = [
  'index.creation_date',
  'index.provided_name',
  'index.uuid',
  'index.version.created',
];

export function IndexSettingsPage({
  connection,
  index,
  notify,
}: {
  connection: HostBodyWritable;
  index: string;
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
      const flat = flattenIndexSettings(result.data.data, index);
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
      if (value === settings[name]) delete next[name];
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

  const groups = useMemo(
    () =>
      groupSettings(
        Object.keys(form)
          .filter(isValidIndexSetting)
          .map((name) => ({ name, static: !dynamicIndexSettings.has(name) })),
      ),
    [form],
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
    <SplitPane
      storageKey="cerebro.indexSettingsSplitPercent"
      leftMinWidth={520}
      left={
        <>
          <div className="row query-container">
            <div className="col-lg-4 col-md-6 col-sm-6 col-xs-12">
              <input
                className="form-control"
                placeholder="filter settings by name"
                value={filter.name}
                onChange={(event) => setFilter((value) => ({ ...value, name: event.target.value }))}
              />
            </div>
            <div className="col-lg-4 col-md-6 col-sm-6 col-xs-12">
              <div className="checkbox">
                <label>
                  <input
                    checked={filter.showStatic}
                    type="checkbox"
                    onChange={(event) => setFilter((value) => ({ ...value, showStatic: event.target.checked }))}
                  />{' '}
                  show static settings <Icon className="alert-warning" name="lock" />
                </label>
              </div>
            </div>
          </div>
          {visibleGroups.map((group) => (
            <div className="row form-group" key={group.name}>
              <div className="col-xs-12">
                <h6>
                  <b>{group.name.toUpperCase()}</b>
                </h6>
                <hr className="header" />
              </div>
              {group.settings.map((setting) => (
                <div className="col-lg-4 col-md-4 col-sm-6 col-xs-12" key={setting.name}>
                  <div className="form-group">
                    <label className="form-label">
                      {setting.name} {setting.static ? <Icon className="alert-warning" name="lock" /> : null}
                    </label>
                    <input
                      className="form-control"
                      disabled={setting.static}
                      type="text"
                      value={form[setting.name] ?? ''}
                      onChange={(event) => updateSetting(setting.name, event.target.value)}
                    />
                  </div>
                </div>
              ))}
            </div>
          ))}
        </>
      }
      right={
        <>
          <h4>
            pending changes <small className="info-text">({pending})</small>
          </h4>
          {pending ? (
            <>
              <table className="table table-condensed">
                <tbody>
                  {Object.entries(changes).map(([setting, value]) => (
                    <tr key={setting}>
                      <td>
                        <Icon name="cog" /> {setting} <span className="info-text">updated to</span> {value}
                      </td>
                      <td className="text-right">
                        <button className="btn btn-default btn-xs" title="undo" type="button" onClick={() => updateSetting(setting, settings[setting] ?? '')}>
                          <Icon name="undo" />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <div className="text-right">
                <button className="btn btn-success" type="button" onClick={() => void save()}>
                  save
                </button>
              </div>
            </>
          ) : (
            <div className="info-text">no pending changes</div>
          )}
        </>
      }
    />
  );
}

function flattenIndexSettings(raw: unknown, index: string): Record<string, string> {
  const root = raw as Record<string, unknown>;
  const indexSettings = (root[index] ?? root) as { defaults?: unknown; settings?: unknown };
  const result: Record<string, string> = {};
  flatten(indexSettings.defaults, '', result);
  flatten(indexSettings.settings, '', result);
  return result;
}

function flatten(value: unknown, prefix: string, out: Record<string, string>) {
  if (value && typeof value === 'object' && !Array.isArray(value)) {
    Object.entries(value as Record<string, unknown>).forEach(([key, nested]) => {
      flatten(nested, prefix ? `${prefix}.${key}` : key, out);
    });
    return;
  }
  if (prefix) out[prefix] = textValue(value);
}

function isValidIndexSetting(setting: string) {
  return invalidIndexSettings.every((invalid) => !setting.includes(invalid));
}

function groupSettings(settings: Setting[]) {
  const groups = new Map<string, Setting[]>();
  settings.forEach((setting) => {
    const group = setting.name.split('.')[1] || setting.name.split('.')[0] || 'settings';
    groups.set(group, [...(groups.get(group) ?? []), setting]);
  });
  return Array.from(groups, ([name, groupSettings]) => ({
    name,
    settings: groupSettings.sort((a, b) => a.name.localeCompare(b.name)),
  })).sort((a, b) => a.name.localeCompare(b.name));
}
