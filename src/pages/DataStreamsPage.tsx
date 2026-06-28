import { useEffect, useState, type ReactNode } from 'react';
import { Link } from '@tanstack/react-router';

import {
  dataStreamsAttachIlm,
  dataStreamsCreate,
  dataStreamsDelete,
  dataStreamsDetachIlm,
  dataStreamsList,
  dataStreamsRollover,
  dataStreamsUpdateLifecycle,
} from '../api/dataStreamsClient';
import { ilmPoliciesList } from '../api/ilmClient';
import type { DataStream, HostBodyWritable, IlmPolicy } from '../api/client/types.gen';
import { Icon } from '../components/Icon';
import { ConfirmModal, ModalFrame, useEscape } from '../components/Modal';
import { SplitPane } from '../components/SplitPane';
import type { Notify } from '../types';
import { errorMessage, formatBytes, formatJson, textValue, timeInterval } from '../utils/format';
import { nextSort, sortByText, type SortState } from '../utils/sort';

type DataStreamSortKey = 'name' | 'status' | 'generation' | 'backing_indices' | 'size';
type RetentionMode = 'retention' | 'infinite' | 'disabled';

export function DataStreamsPage({
  connection,
  notify,
  refreshTick,
}: {
  connection: HostBodyWritable;
  notify: Notify;
  refreshTick: number;
}) {
  const [streams, setStreams] = useState<DataStream[]>([]);
  const [supported, setSupported] = useState(true);
  const [selectedName, setSelectedName] = useState('');
  const [filter, setFilter] = useState('');
  const [sort, setSort] = useState<SortState<DataStreamSortKey>>({ key: 'name', order: 'asc' });
  const [rolloverConfirm, setRolloverConfirm] = useState<DataStream | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<DataStream | null>(null);
  const [retentionStream, setRetentionStream] = useState<DataStream | null>(null);
  const [attachStream, setAttachStream] = useState<DataStream | null>(null);
  const [detachStream, setDetachStream] = useState<DataStream | null>(null);
  const [policies, setPolicies] = useState<IlmPolicy[]>([]);
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    void load();
  }, [connection, refreshTick]);

  async function load() {
    try {
      const result = await dataStreamsList<true>({ body: connection, throwOnError: true });
      const items = result.data.items ?? [];
      setSupported(result.data.supported);
      setStreams(items);
      setSelectedName((current) => current && items.some((stream) => stream.name === current) ? current : (items[0]?.name ?? ''));
    } catch (error) {
      notify('danger', `Error loading data streams: ${errorMessage(error)}`);
    }
  }

  async function rollover(stream: DataStream) {
    try {
      await dataStreamsRollover<true>({ body: { ...connection, name: stream.name }, throwOnError: true });
      notify('info', `Data stream ${stream.name} rolled over`);
      await load();
    } catch (error) {
      notify('danger', `Error rolling over data stream: ${errorMessage(error)}`);
    }
  }

  async function create(name: string) {
    try {
      await dataStreamsCreate<true>({ body: { ...connection, name }, throwOnError: true });
      notify('info', `Data stream ${name} created`);
      setCreateOpen(false);
      setSelectedName(name);
      await load();
    } catch (error) {
      notify('danger', `Error creating data stream: ${errorMessage(error)}`);
    }
  }

  async function remove(stream: DataStream) {
    try {
      await dataStreamsDelete<true>({ body: { ...connection, name: stream.name }, throwOnError: true });
      notify('info', `Data stream ${stream.name} deleted`);
      setSelectedName('');
      await load();
    } catch (error) {
      notify('danger', `Error deleting data stream: ${errorMessage(error)}`);
    }
  }

  async function saveLifecycle(stream: DataStream, lifecycle: unknown) {
    try {
      await dataStreamsUpdateLifecycle<true>({
        body: { ...connection, lifecycle, name: stream.name },
        throwOnError: true,
      });
      notify('info', `Data stream ${stream.name} lifecycle updated`);
      setRetentionStream(null);
      await load();
    } catch (error) {
      notify('danger', `Error updating data stream lifecycle: ${errorMessage(error)}`);
    }
  }

  async function openAttachILM(stream: DataStream) {
    try {
      const result = await ilmPoliciesList<true>({ body: connection, throwOnError: true });
      setPolicies((result.data.items ?? []).sort((left, right) => left.name.localeCompare(right.name)));
      setAttachStream(stream);
    } catch (error) {
      notify('danger', `Error loading ILM policies: ${errorMessage(error)}`);
    }
  }

  async function attachILM(stream: DataStream, policy: string, updateBackingIndices: boolean, rolloverAfterAttach: boolean) {
    try {
      await dataStreamsAttachIlm<true>({
        body: {
          ...connection,
          name: stream.name,
          policy,
          rollover: rolloverAfterAttach,
          update_backing_indices: updateBackingIndices,
        },
        throwOnError: true,
      });
      notify('info', `ILM policy ${policy} attached to ${stream.name}`);
      setAttachStream(null);
      await load();
    } catch (error) {
      notify('danger', `Error attaching ILM policy: ${errorMessage(error)}`);
    }
  }

  async function detachILM(stream: DataStream, updateBackingIndices: boolean) {
    try {
      await dataStreamsDetachIlm<true>({
        body: {
          ...connection,
          name: stream.name,
          update_backing_indices: updateBackingIndices,
        },
        throwOnError: true,
      });
      notify('info', `ILM detached from ${stream.name}`);
      setDetachStream(null);
      await load();
    } catch (error) {
      notify('danger', `Error detaching ILM policy: ${errorMessage(error)}`);
    }
  }

  const filtered = sortByText(
    streams.filter((stream) => stream.name.toLowerCase().includes(filter.toLowerCase())),
    sort,
    dataStreamSortValue,
  );
  const selected = streams.find((stream) => stream.name === selectedName) ?? filtered[0] ?? null;

  if (!supported) {
    return (
      <div className="alert alert-info">
        <Icon name="info" /> Data streams are not supported by this Elasticsearch version.
      </div>
    );
  }

  return (
    <>
      {rolloverConfirm ? (
        <ConfirmModal
          body={
            <>
              Rollover data stream <strong>{rolloverConfirm.name}</strong>? Elasticsearch will create a new write backing index when rollover conditions allow it.
            </>
          }
          confirmClassName="btn-warning"
          confirmLabel={
            <>
              <Icon name="refresh" /> rollover
            </>
          }
          onClose={() => setRolloverConfirm(null)}
          onConfirm={() => rollover(rolloverConfirm)}
          title="rollover data stream"
        />
      ) : null}
      {deleteConfirm ? (
        <ConfirmModal
          body={
            <>
              Delete data stream <strong>{deleteConfirm.name}</strong> and all backing indices? This operation cannot be undone.
            </>
          }
          confirmLabel={
            <>
              <Icon name="trash" /> delete data stream
            </>
          }
          onClose={() => setDeleteConfirm(null)}
          onConfirm={() => remove(deleteConfirm)}
          title="delete data stream"
        />
      ) : null}
      {createOpen ? (
        <CreateDataStreamModal onClose={() => setCreateOpen(false)} onCreate={create} />
      ) : null}
      {retentionStream ? (
        <RetentionModal
          stream={retentionStream}
          onClose={() => setRetentionStream(null)}
          onSave={(lifecycle) => saveLifecycle(retentionStream, lifecycle)}
        />
      ) : null}
      {attachStream ? (
        <AttachILMModal
          policies={policies}
          stream={attachStream}
          onAttach={(policy, updateBackingIndices, rolloverAfterAttach) => attachILM(attachStream, policy, updateBackingIndices, rolloverAfterAttach)}
          onClose={() => setAttachStream(null)}
        />
      ) : null}
      {detachStream ? (
        <DetachILMModal
          stream={detachStream}
          onClose={() => setDetachStream(null)}
          onDetach={(updateBackingIndices) => detachILM(detachStream, updateBackingIndices)}
        />
      ) : null}
      <SplitPane
        storageKey="cerebro.dataStreamsSplitPercent"
        left={
          <>
            <h4>
              data streams <small className="info-text">({filtered.length})</small>
            </h4>
            <div className="form-group">
              <button className="btn btn-success" type="button" onClick={() => setCreateOpen(true)}>
                <Icon name="plus" /> create data stream
              </button>
            </div>
            <div className="row">
              <div className="col-xs-12 form-group">
                <input className="form-control" placeholder="filter data streams by name" value={filter} onChange={(event) => setFilter(event.target.value)} />
              </div>
              <div className="col-xs-12">
                {filtered.length ? (
                  <table className="table table-condensed">
                    <thead>
                      <tr>
                        <th>{sortButton('name', 'name', sort, setSort)}</th>
                        <th>{sortButton('status', 'status', sort, setSort)}</th>
                        <th>{sortButton('generation', 'gen', sort, setSort)}</th>
                        <th>{sortButton('backing_indices', 'backing', sort, setSort)}</th>
                        <th>{sortButton('size', 'size', sort, setSort)}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {filtered.map((stream) => (
                        <tr
                          className={`cursor-pointer ${selected?.name === stream.name ? 'bg-[#434749]' : ''}`}
                          key={stream.name}
                          onClick={() => setSelectedName(stream.name)}
                        >
                          <td className="normal-action">
                            <Icon name="database" /> {stream.name}
                            {stream.hidden ? <span className="label ml-2 bg-[#8b8f95]">hidden</span> : null}
                            {stream.system ? <span className="label ml-2 bg-[#e4d836]">system</span> : null}
                          </td>
                          <td><HealthLabel status={stream.status} /></td>
                          <td>{stream.generation}</td>
                          <td>{stream.backing_indices_count}</td>
                          <td>{formatBytes(stream.store_size_bytes)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                ) : (
                  <div className="info-text">no data streams match current filter</div>
                )}
              </div>
            </div>
          </>
        }
        right={
          selected ? (
            <DataStreamDetails
              connection={connection}
              stream={selected}
              onAttachILM={() => openAttachILM(selected)}
              onDelete={() => setDeleteConfirm(selected)}
              onDetachILM={() => setDetachStream(selected)}
              onEditRetention={() => setRetentionStream(selected)}
              onRollover={() => setRolloverConfirm(selected)}
            />
          ) : (
            <div className="info-text">select a data stream</div>
          )
        }
      />
    </>
  );
}

function DataStreamDetails({
  connection,
  onAttachILM,
  onDelete,
  onDetachILM,
  onEditRetention,
  onRollover,
  stream,
}: {
  connection: HostBodyWritable;
  onAttachILM: () => void;
  onDelete: () => void;
  onDetachILM: () => void;
  onEditRetention: () => void;
  onRollover: () => void;
  stream: DataStream;
}) {
  const writeIndex = (stream.backing_indices ?? []).find((index) => index.write_index);
  const lifecycle = lifecycleSummary(stream.lifecycle);
  const managedBy = dataStreamManagedBy(stream);
  const retentionEditable = canEditDataStreamLifecycle(stream);
  const ilmPolicy = writeIndex?.ilm_policy || '';

  return (
    <>
      <div className="flex items-center justify-between gap-[15px]">
        <h4 className="m-0 break-all">
          <Icon name="database" /> {stream.name}
        </h4>
        <div className="inline-flex items-center gap-[10px]">
          <button className="btn btn-default btn-xs" disabled={!stream.template} title={stream.template ? 'attach ILM policy' : 'data stream has no template'} type="button" onClick={onAttachILM}>
            <Icon name="history" /> attach ilm
          </button>
          <button className="btn btn-default btn-xs" disabled={!ilmPolicy || !stream.template} title={ilmPolicy ? 'detach ILM policy' : 'no ILM policy attached'} type="button" onClick={onDetachILM}>
            <Icon name="undo" /> detach ilm
          </button>
          <button
            className="btn btn-default btn-xs"
            disabled={!retentionEditable}
            title={retentionEditable ? 'edit data stream lifecycle retention' : 'retention is managed by ILM policy'}
            type="button"
            onClick={onEditRetention}
          >
            <Icon name="pencil" /> retention
          </button>
          <button className="btn btn-warning btn-xs" title="rollover" type="button" onClick={onRollover}>
            <Icon name="refresh" /> rollover
          </button>
          <button
            className="btn btn-danger btn-xs"
            disabled={stream.system}
            title={stream.system ? 'system data streams should not be deleted from Cerebro' : 'delete data stream'}
            type="button"
            onClick={onDelete}
          >
            <Icon name="trash" />
          </button>
        </div>
      </div>
      <div className="subtitle">
        timestamp: <span className="text-[#eceeef]">{stream.timestamp_field || '@timestamp'}</span>{' '}
        | template: <span className="text-[#eceeef]">{stream.template || 'none'}</span>
      </div>
      <div className="row mt-[15px]">
        <InfoBox label="status" value={<HealthLabel status={stream.status} />} />
        <InfoBox label="generation" value={stream.generation} />
        <InfoBox label="backing indices" value={stream.backing_indices_count} />
        <InfoBox label="store size" value={formatBytes(stream.store_size_bytes)} />
      </div>
      <div className="row">
        <InfoBox label="write index" value={writeIndex?.name ?? 'none'} />
        <InfoBox label="max timestamp" value={formatTimestamp(stream.maximum_timestamp)} />
        <InfoBox label="managed by" value={managedBy} />
        <InfoBox label="next generation" value={stream.next_generation_managed_by || 'same'} />
      </div>
      <div className="row">
        <InfoBox label="data stream lifecycle" value={lifecycle} />
        <InfoBox
          label="ilm policy"
          value={
            ilmPolicy ? (
              <Link className="normal-action" search={{ host: connection.host, policy: ilmPolicy }} to="/ilm">
                {ilmPolicy}
              </Link>
            ) : (
              'none'
            )
          }
        />
        <InfoBox label="flags" value={streamFlags(stream)} />
        <InfoBox label="rollover" value={stream.rollover_on_write ? 'on write' : 'manual / policy'} />
      </div>

      <div className="mt-[15px] flex items-center justify-between">
        <h4 className="m-0">backing indices</h4>
        <Link className="btn btn-default btn-xs" search={{ host: connection.host, index: stream.name }} to="/data_explorer">
          <Icon name="database" /> browse data
        </Link>
      </div>
      <table className="table table-condensed">
        <thead>
          <tr>
            <th>index</th>
            <th>health</th>
            <th>docs</th>
            <th>size</th>
            <th>managed by</th>
            <th>lifecycle</th>
            <th className="text-right">actions</th>
          </tr>
        </thead>
        <tbody>
          {(stream.backing_indices ?? []).map((index) => (
            <tr key={index.name}>
              <td className="break-all">
                {index.write_index ? <span className="label label-success mr-2">write</span> : null}
                {index.name}
              </td>
              <td><HealthLabel status={index.health} /></td>
              <td>{textValue(index.docs_count) || '0'}</td>
              <td>{formatBytes(index.store_size_bytes)}</td>
              <td>{index.managed_by || 'unmanaged'}</td>
              <td>
                {index.ilm_managed ? (
                  <>
                    {index.ilm_policy ? (
                      <div>
                        <span className="info-text">policy:</span>{' '}
                        <Link className="normal-action" search={{ host: connection.host, policy: index.ilm_policy }} to="/ilm">
                          {index.ilm_policy}
                        </Link>
                      </div>
                    ) : null}
                    <div>{[index.ilm_phase, index.ilm_action, index.ilm_step].filter(Boolean).join(' / ') || 'managed'}</div>
                  </>
                ) : (
                  <span className="info-text">not ILM managed</span>
                )}
              </td>
              <td className="text-right">
                <span className="inline-flex items-center gap-[10px]">
                  <Link className="btn btn-default btn-xs" search={{ host: connection.host, index: index.name }} title="index settings" to="/index_settings">
                    <Icon name="cog" />
                  </Link>
                  <Link className="btn btn-default btn-xs" search={{ host: connection.host, index: index.name }} title="browse backing index data" to="/data_explorer">
                    <Icon name="database" />
                  </Link>
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <details>
        <summary className="normal-action info-text">raw lifecycle</summary>
        <pre>{formatJson(stream.lifecycle ?? {})}</pre>
      </details>
    </>
  );
}

function CreateDataStreamModal({
  onClose,
  onCreate,
}: {
  onClose: () => void;
  onCreate: (name: string) => Promise<void> | void;
}) {
  const [name, setName] = useState('');
  useEscape(onClose);
  const normalized = name.trim();

  return (
    <ModalFrame dialogClassName="" onClose={onClose} title="create data stream">
      <div className="modal-body">
        <div className="alert alert-info">
          <Icon name="info" /> Elasticsearch requires a matching index template with data_stream enabled before a data stream can be created.
        </div>
        <div className="form-group">
          <label className="form-label">data stream name</label>
          <input className="form-control font-mono" placeholder="logs-app" value={name} onChange={(event) => setName(event.target.value)} />
        </div>
      </div>
      <div className="modal-footer">
        <button className="btn btn-default" type="button" onClick={onClose}>
          cancel
        </button>
        <button className="btn btn-success" disabled={!normalized} type="button" onClick={() => void onCreate(normalized)}>
          <Icon name="plus" /> create
        </button>
      </div>
    </ModalFrame>
  );
}

function RetentionModal({
  onClose,
  onSave,
  stream,
}: {
  onClose: () => void;
  onSave: (lifecycle: unknown) => Promise<void> | void;
  stream: DataStream;
}) {
  const currentRetention = lifecycleRetention(stream.lifecycle);
  const [mode, setMode] = useState<RetentionMode>(currentRetention === false ? 'disabled' : currentRetention ? 'retention' : 'infinite');
  const [retention, setRetention] = useState(typeof currentRetention === 'string' ? currentRetention : '30d');
  useEscape(onClose);

  function lifecycleBody() {
    if (mode === 'disabled') return { enabled: false };
    if (mode === 'infinite') return {};
    return { data_retention: retention.trim() };
  }

  return (
    <ModalFrame dialogClassName="" onClose={onClose} title={`edit retention: ${stream.name}`}>
      <div className="modal-body">
        <div className="form-group">
          <label className="form-label">retention mode</label>
          <select className="form-control" value={mode} onChange={(event) => setMode(event.target.value as RetentionMode)}>
            <option value="retention">keep data for</option>
            <option value="infinite">no data stream retention</option>
            <option value="disabled">disable data stream lifecycle</option>
          </select>
        </div>
        {mode === 'retention' ? (
          <div className="form-group">
            <label className="form-label">retention</label>
            <input className="form-control font-mono" placeholder="30d, 90d, 12h" value={retention} onChange={(event) => setRetention(event.target.value)} />
          </div>
        ) : null}
        <div className="subtitle">request body</div>
        <pre>{formatJson(lifecycleBody())}</pre>
      </div>
      <div className="modal-footer">
        <button className="btn btn-default" type="button" onClick={onClose}>
          cancel
        </button>
        <button
          className="btn btn-success"
          disabled={mode === 'retention' && !retention.trim()}
          type="button"
          onClick={() => void onSave(lifecycleBody())}
        >
          <Icon name="save" /> save
        </button>
      </div>
    </ModalFrame>
  );
}

function AttachILMModal({
  onAttach,
  onClose,
  policies,
  stream,
}: {
  onAttach: (policy: string, updateBackingIndices: boolean, rolloverAfterAttach: boolean) => Promise<void> | void;
  onClose: () => void;
  policies: IlmPolicy[];
  stream: DataStream;
}) {
  const currentPolicy = stream.backing_indices?.find((index) => index.write_index)?.ilm_policy ?? '';
  const [policy, setPolicy] = useState(currentPolicy || policies[0]?.name || '');
  const [updateBackingIndices, setUpdateBackingIndices] = useState(true);
  const [rolloverAfterAttach, setRolloverAfterAttach] = useState(false);
  useEscape(onClose);

  return (
    <ModalFrame dialogClassName="" onClose={onClose} title={`attach ilm: ${stream.name}`}>
      <div className="modal-body">
        <div className="form-group">
          <label className="form-label">policy</label>
          <select className="form-control" value={policy} onChange={(event) => setPolicy(event.target.value)}>
            {policies.map((item) => (
              <option key={item.name} value={item.name}>
                {item.name}
              </option>
            ))}
          </select>
        </div>
        <div className="checkbox">
          <label>
            <input checked={updateBackingIndices} type="checkbox" onChange={(event) => setUpdateBackingIndices(event.target.checked)} /> update existing backing indices
          </label>
        </div>
        <div className="checkbox">
          <label>
            <input checked={rolloverAfterAttach} type="checkbox" onChange={(event) => setRolloverAfterAttach(event.target.checked)} /> rollover after attach
          </label>
        </div>
        <div className="subtitle">
          template: <span className="text-[#eceeef]">{stream.template || 'none'}</span>
        </div>
        {!policies.length ? (
          <div className="alert alert-warning mt-[15px]">
            <Icon name="warning" /> no ILM policies available
          </div>
        ) : null}
      </div>
      <div className="modal-footer">
        <button className="btn btn-default" type="button" onClick={onClose}>
          cancel
        </button>
        <button className="btn btn-success" disabled={!policy || !policies.length} type="button" onClick={() => void onAttach(policy, updateBackingIndices, rolloverAfterAttach)}>
          <Icon name="history" /> attach
        </button>
      </div>
    </ModalFrame>
  );
}

function DetachILMModal({
  onClose,
  onDetach,
  stream,
}: {
  onClose: () => void;
  onDetach: (updateBackingIndices: boolean) => Promise<void> | void;
  stream: DataStream;
}) {
  const [updateBackingIndices, setUpdateBackingIndices] = useState(true);
  useEscape(onClose);

  return (
    <ModalFrame dialogClassName="" onClose={onClose} title={`detach ilm: ${stream.name}`}>
      <div className="modal-body">
        <div className="checkbox">
          <label>
            <input checked={updateBackingIndices} type="checkbox" onChange={(event) => setUpdateBackingIndices(event.target.checked)} /> update existing backing indices
          </label>
        </div>
        <div className="subtitle">
          template: <span className="text-[#eceeef]">{stream.template || 'none'}</span>
        </div>
      </div>
      <div className="modal-footer">
        <button className="btn btn-default" type="button" onClick={onClose}>
          cancel
        </button>
        <button className="btn btn-warning" type="button" onClick={() => void onDetach(updateBackingIndices)}>
          <Icon name="undo" /> detach
        </button>
      </div>
    </ModalFrame>
  );
}

function sortButton(
  key: DataStreamSortKey,
  label: string,
  sort: SortState<DataStreamSortKey>,
  setSort: (value: SortState<DataStreamSortKey> | ((value: SortState<DataStreamSortKey>) => SortState<DataStreamSortKey>)) => void,
) {
  return (
    <button className="normal-action border-0 bg-transparent p-0 text-inherit" type="button" onClick={() => setSort((value) => nextSort(value, key))}>
      {label} {sort.key === key ? <Icon name={sort.order === 'asc' ? 'caret-down' : 'sort-alpha-desc'} /> : null}
    </button>
  );
}

function dataStreamSortValue(stream: DataStream, key: DataStreamSortKey) {
  switch (key) {
    case 'status':
      return stream.status ?? '';
    case 'generation':
      return String(stream.generation).padStart(12, '0');
    case 'backing_indices':
      return String(stream.backing_indices_count).padStart(12, '0');
    case 'size':
      return String(stream.store_size_bytes).padStart(20, '0');
    default:
      return stream.name;
  }
}

function HealthLabel({ status }: { status?: string }) {
  const normalized = (status ?? '').toLowerCase();
  const className =
    normalized === 'green'
      ? 'label label-success'
      : normalized === 'yellow'
        ? 'label label-warning'
        : normalized === 'red'
          ? 'label label-danger'
          : 'label bg-[#8b8f95]';
  return <span className={className}>{status || 'unknown'}</span>;
}

function InfoBox({ label, value }: { label: string; value: unknown }) {
  return (
    <div className="col-sm-3 form-group">
      <div className="border border-[#55595c] px-3 py-2">
        <div className="text-[11px] uppercase text-[#8b8f95]">{label}</div>
        <div className="truncate" title={typeof value === 'string' ? value : undefined}>{value as ReactNode}</div>
      </div>
    </div>
  );
}

function lifecycleSummary(lifecycle: unknown) {
  const retention = lifecycleRetention(lifecycle);
  if (retention === false) return 'disabled';
  if (typeof retention === 'string') return retention;
  if (lifecycle && Object.keys(lifecycle as Record<string, unknown>).length > 0) return 'configured';
  return 'none';
}

function lifecycleRetention(lifecycle: unknown): string | false | undefined {
  if (!lifecycle || typeof lifecycle !== 'object') return undefined;
  const body = lifecycle as { data_retention?: unknown; enabled?: unknown };
  if (body.enabled === false) return false;
  const retention = textValue(body.data_retention);
  return retention || undefined;
}

function canEditDataStreamLifecycle(stream: DataStream) {
  return !isILMManaged(stream);
}

function isILMManaged(stream: DataStream) {
  const managedBy = dataStreamManagedBy(stream).toLowerCase();
  return managedBy.includes('index lifecycle') || managedBy === 'ilm' || (stream.prefer_ilm && stream.backing_indices?.some((index) => index.ilm_managed));
}

function dataStreamManagedBy(stream: DataStream) {
  if (stream.managed_by) return stream.managed_by;
  const writeIndex = stream.backing_indices?.find((index) => index.write_index);
  if (writeIndex?.managed_by) return writeIndex.managed_by;
  if (stream.prefer_ilm && writeIndex?.ilm_managed) return 'Index Lifecycle Management';
  return stream.lifecycle && Object.keys(stream.lifecycle as Record<string, unknown>).length > 0 ? 'Data stream lifecycle' : 'Unmanaged';
}

function streamFlags(stream: DataStream) {
  const flags = [
    stream.hidden ? 'hidden' : '',
    stream.system ? 'system' : '',
    stream.prefer_ilm ? 'prefer ilm' : '',
    stream.rollover_on_write ? 'rollover on write' : '',
  ].filter(Boolean);
  return flags.length ? flags.join(', ') : 'none';
}

function formatTimestamp(value: unknown) {
  const ms = Number(value);
  if (!Number.isFinite(ms) || ms <= 0) return 'none';
  return `${new Date(ms).toISOString()} (${timeInterval(Date.now() - ms)} ago)`;
}
