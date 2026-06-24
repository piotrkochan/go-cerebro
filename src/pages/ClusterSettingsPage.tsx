import { useEffect, useMemo, useState } from 'react';

import { clusterSettingsGet, clusterSettingsSave, type HostBodyWritable } from '../api/client';
import { Icon } from '../components/Icon';
import type { Notify } from '../types';
import { errorMessage, textValue } from '../utils/format';

type Change = { transient: boolean; value: string };
type Setting = { name: string; static: boolean };

const dynamicSettings = new Set([
  'action.auto_create_index',
  'action.destructive_requires_name',
  'action.search.shard_count.limit',
  'cluster.blocks.read_only',
  'cluster.blocks.read_only_allow_delete',
  'cluster.indices.close.enable',
  'cluster.info.update.interval',
  'cluster.info.update.timeout',
  'cluster.routing.allocation.allow_rebalance',
  'cluster.routing.allocation.awareness.attributes',
  'cluster.routing.allocation.balance.index',
  'cluster.routing.allocation.balance.shard',
  'cluster.routing.allocation.balance.threshold',
  'cluster.routing.allocation.cluster_concurrent_rebalance',
  'cluster.routing.allocation.disk.include_relocations',
  'cluster.routing.allocation.disk.reroute_interval',
  'cluster.routing.allocation.disk.threshold_enabled',
  'cluster.routing.allocation.disk.watermark.high',
  'cluster.routing.allocation.disk.watermark.low',
  'cluster.routing.allocation.enable',
  'cluster.routing.allocation.node_concurrent_incoming_recoveries',
  'cluster.routing.allocation.node_concurrent_outgoing_recoveries',
  'cluster.routing.allocation.node_concurrent_recoveries',
  'cluster.routing.allocation.node_initial_primaries_recoveries',
  'cluster.routing.allocation.same_shard.host',
  'cluster.routing.allocation.snapshot.relocation_enabled',
  'cluster.routing.allocation.total_shards_per_node',
  'cluster.routing.rebalance.enable',
  'cluster.service.slow_task_logging_threshold',
  'discovery.zen.commit_timeout',
  'discovery.zen.minimum_master_nodes',
  'discovery.zen.no_master_block',
  'discovery.zen.publish_diff.enable',
  'discovery.zen.publish_timeout',
  'gateway.initial_shards',
  'indices.breaker.fielddata.limit',
  'indices.breaker.fielddata.overhead',
  'indices.breaker.request.limit',
  'indices.breaker.request.overhead',
  'indices.breaker.total.limit',
  'indices.mapping.dynamic_timeout',
  'indices.recovery.internal_action_long_timeout',
  'indices.recovery.internal_action_timeout',
  'indices.recovery.max_bytes_per_sec',
  'indices.recovery.recovery_activity_timeout',
  'indices.recovery.retry_delay_network',
  'indices.recovery.retry_delay_state_sync',
  'indices.store.throttle.max_bytes_per_sec',
  'indices.store.throttle.type',
  'indices.ttl.interval',
  'ingest.new_date_format',
  'network.breaker.inflight_requests.limit',
  'network.breaker.inflight_requests.overhead',
  'script.max_compilations_per_minute',
  'search.default_search_timeout',
  'search.low_level_cancellation',
  'transport.tracer.exclude.0',
  'transport.tracer.exclude.1',
  'xpack.ml.node_concurrent_job_allocations',
  'xpack.monitoring.collection.cluster.state.timeout',
  'xpack.monitoring.collection.cluster.stats.timeout',
  'xpack.monitoring.collection.index.recovery.active_only',
  'xpack.monitoring.collection.index.recovery.timeout',
  'xpack.monitoring.collection.index.stats.timeout',
  'xpack.monitoring.collection.interval',
  'xpack.monitoring.collection.ml.job.stats.timeout',
  'xpack.monitoring.history.duration',
  'xpack.security.http.filter.enabled',
  'xpack.security.transport.filter.enabled',
  'xpack.watcher.history.cleaner_service.enabled',
]);

export function ClusterSettingsPage({
  connection,
  notify,
  refreshTick,
}: {
  connection: HostBodyWritable;
  notify: Notify;
  refreshTick: number;
}) {
  const [settings, setSettings] = useState<Record<string, string>>({});
  const [form, setForm] = useState<Record<string, string>>({});
  const [changes, setChanges] = useState<Record<string, Change>>({});
  const [filter, setFilter] = useState({ name: '', showStatic: false });

  useEffect(() => {
    void load();
  }, [connection, refreshTick]);

  async function load() {
    try {
      const result = await clusterSettingsGet<true>({ body: connection, throwOnError: true });
      const flat = flattenSettings(result.data.data);
      setSettings(flat);
      setForm(flat);
      setChanges({});
    } catch (error) {
      notify('danger', `Error loading cluster settings: ${errorMessage(error)}`);
    }
  }

  function updateSetting(name: string, value: string) {
    setForm((current) => ({ ...current, [name]: value }));
    setChanges((current) => {
      const next = { ...current };
      if (value === settings[name]) {
        delete next[name];
      } else {
        next[name] = { transient: current[name]?.transient ?? true, value };
      }
      return next;
    });
  }

  async function save() {
    const body = { persistent: {} as Record<string, string | null>, transient: {} as Record<string, string | null> };
    Object.entries(changes).forEach(([setting, change]) => {
      body[change.transient ? 'transient' : 'persistent'][setting] = change.value.length > 0 ? change.value : null;
    });
    try {
      await clusterSettingsSave<true>({ body: { ...connection, settings: body }, throwOnError: true });
      notify('info', 'Settings successfully saved');
      await load();
    } catch (error) {
      notify('danger', `Error while saving settings: ${errorMessage(error)}`);
    }
  }

  const groups = useMemo(() => groupSettings(Object.keys(form).map((name) => ({ name, static: !dynamicSettings.has(name) }))), [form]);
  const visibleGroups = groups
    .map((group) => ({
      ...group,
      settings: group.settings.filter((setting) => setting.name.includes(filter.name) && (!setting.static || filter.showStatic)),
    }))
    .filter((group) => group.settings.length > 0);
  const pending = Object.keys(changes).length;

  return (
    <>
      <div className="row query-container">
        <div className="col-lg-4 col-md-6 col-sm-6 col-xs-12">
          <input className="form-control" placeholder="filter settings by name" value={filter.name} onChange={(event) => setFilter((value) => ({ ...value, name: event.target.value }))} />
        </div>
        <div className="col-lg-4 col-md-6 col-sm-6 col-xs-12">
          <div className="checkbox">
            <label>
              <input checked={filter.showStatic} type="checkbox" onChange={(event) => setFilter((value) => ({ ...value, showStatic: event.target.checked }))} /> show static settings <Icon className="alert-warning" name="lock" />
            </label>
          </div>
        </div>
      </div>
      {visibleGroups.map((group) => (
        <div className="row form-group" key={group.name}>
          <div className="col-xs-12">
            <h6><b>{group.name.toUpperCase()}</b></h6>
            <hr className="header" />
          </div>
          {group.settings.map((setting) => (
            <div className="col-lg-4 col-md-4 col-sm-6 col-xs-12" key={setting.name}>
              <div className="form-group">
                <label className="form-label">
                  {setting.name} {setting.static ? <Icon className="alert-warning" name="lock" /> : null}
                </label>
                <input className="form-control" disabled={setting.static} type="text" value={form[setting.name] ?? ''} onChange={(event) => updateSetting(setting.name, event.target.value)} />
              </div>
            </div>
          ))}
        </div>
      ))}
      {pending ? (
        <div className="row" style={{ paddingTop: (pending * 40) + 90 }}>
          <div className="pending-changes">
            <div className="col-xs-12">
              <table className="table">
                <thead>
                  <tr className="text-center"><td>{pending} pending changes</td></tr>
                </thead>
                <tbody>
                  {Object.entries(changes).map(([setting, change]) => (
                    <tr key={setting}>
                      <td>
                        <Icon name="cog" /> {setting} <span className="info-text">updated to</span> {change.value}{' '}
                        <span className="info-text">as</span>{' '}
                        <u className="normal-action" onClick={() => setChanges((value) => ({ ...value, [setting]: { ...change, transient: !change.transient } }))}>
                          {change.transient ? 'transient' : 'persistent'}
                        </u>
                        <Icon className="normal-action pull-right" name="undo" onClick={() => updateSetting(setting, settings[setting] ?? '')} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div className="col-lg-12 text-right">
              <div className="form-group">
                <div className="btn-group">
                  <button className="btn btn-success" onClick={() => void save()}>save</button>
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
}

function flattenSettings(raw: unknown): Record<string, string> {
  const result: Record<string, string> = {};
  const root = raw as { defaults?: unknown; persistent?: unknown; transient?: unknown };
  ['defaults', 'persistent', 'transient'].forEach((section) => flatten(root?.[section as keyof typeof root], '', result));
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

function groupSettings(settings: Setting[]) {
  const groups = new Map<string, Setting[]>();
  settings.forEach((setting) => {
    const group = setting.name.split('.')[1] || setting.name.split('.')[0] || 'settings';
    groups.set(group, [...(groups.get(group) ?? []), setting]);
  });
  return Array.from(groups, ([name, groupSettings]) => ({ name, settings: groupSettings.sort((a, b) => a.name.localeCompare(b.name)) })).sort((a, b) => a.name.localeCompare(b.name));
}
