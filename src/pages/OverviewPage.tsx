import { useEffect, useState } from 'react';
import { useStore } from '@tanstack/react-store';

import {
  commonsGetIndexMapping,
  commonsGetIndexSettings,
  commonsGetIndexStats,
  overview,
  overviewClearIndicesCache,
  overviewCloseIndices,
  overviewDeleteIndices,
  overviewDisableShardAllocation,
  overviewEnableShardAllocation,
  overviewFlushIndices,
  overviewForceMerge,
  overviewOpenIndices,
  overviewRefreshIndices,
  overviewRelocateShard,
  overviewShardStats,
  type HostBodyWritable,
  type Overview,
  type OverviewIndex,
  type OverviewNode,
} from '../api/client';
import { Checkbox } from '../components/Checkbox';
import { Icon } from '../components/Icon';
import { LazyJsonEditor } from '../components/LazyJsonEditor';
import {
  IndexHeader,
  Loading,
  OverviewNodeCell,
  Pagination,
  type ShardRef,
  Stats,
  byName,
  renderShards,
} from '../components/LegacyUi';
import { ConfirmModal, ModalFrame, useEscape } from '../components/Modal';
import { sessionStore } from '../stores/sessionStore';
import type { Notify } from '../types';
import { clusterPath } from '../utils/connection';
import { errorMessage, formatJson, formatNumber, numberValue, textValue } from '../utils/format';

type JsonDialog = {
  body: unknown;
  title: string;
};

type ConfirmDialog = {
  body: string;
  confirmLabel: string;
  onConfirm: () => Promise<void> | void;
  title: string;
};

export function OverviewPage({
  connection,
  notify,
  refreshTick,
  setStatus,
}: {
  connection: HostBodyWritable;
  notify: Notify;
  refreshTick: number;
  setStatus: (status: string) => void;
}) {
  const [data, setData] = useState<Overview>();
  const [filterName, setFilterName] = useState('');
  const [nodeFilter, setNodeFilter] = useState('');
  const [showClosed, setShowClosed] = useState(false);
  const [showSpecial, setShowSpecial] = useState(false);
  const [expandedView, setExpandedView] = useState(false);
  const [asc, setAsc] = useState(true);
  const [showOnlyAffected, setShowOnlyAffected] = useState(false);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(getOverviewPageSize);
  const [jsonDialog, setJsonDialog] = useState<JsonDialog | null>(null);
  const [confirmDialog, setConfirmDialog] = useState<ConfirmDialog | null>(null);
  const [relocatingShard, setRelocatingShard] = useState<ShardRef | null>(null);
  const dataExplorerEnabled = useStore(sessionStore, (state) => state.features.dataExplorer);

  useEffect(() => {
    let ignore = false;
    void loadOverview((nextData) => {
      if (!ignore) {
        setData(nextData);
        setStatus(textValue(nextData.status));
      }
    });
    return () => {
      ignore = true;
    };
  }, [connection, notify, refreshTick, setStatus]);

  async function loadOverview(onData?: (nextData: Overview) => void) {
    try {
      const result = await overview<true>({ path: clusterPath(connection), throwOnError: true });
      if (onData) onData(result.data);
      else {
        setData(result.data);
        setStatus(textValue(result.data.status));
      }
    } catch (error) {
      notify('danger', errorMessage(error));
    }
  }

  useEffect(() => {
    function resize() {
      setPageSize(getOverviewPageSize());
    }

    window.addEventListener('resize', resize);
    return () => window.removeEventListener('resize', resize);
  }, []);

  async function runAction(label: string, action: () => Promise<unknown>) {
    try {
      await action();
      notify('success', label);
      await loadOverview();
    } catch (error) {
      notify('danger', errorMessage(error));
    }
  }

  async function showJson(title: string, action: () => Promise<{ data: { data: unknown } }>) {
    try {
      const result = await action();
      setJsonDialog({ body: result.data.data, title });
    } catch (error) {
      notify('danger', errorMessage(error));
    }
  }

  function confirmDelete(title: string, body: string, onConfirm: () => Promise<void> | void) {
    setConfirmDialog({ body, confirmLabel: 'delete', onConfirm, title });
  }

  const indexActions = {
    clearCache: (index: OverviewIndex) =>
      void runAction('index cache cleared', () =>
        overviewClearIndicesCache<true>({ path: { ...clusterPath(connection), indices: index.name }, throwOnError: true }),
      ),
    closeIndex: (index: OverviewIndex) =>
      void runAction('index closed', () =>
        overviewCloseIndices<true>({ path: { ...clusterPath(connection), indices: index.name }, throwOnError: true }),
      ),
    deleteIndex: (index: OverviewIndex) =>
      confirmDelete(
        `Delete ${index.name}?`,
        `Delete index ${index.name}? This operation cannot be undone.`,
        () =>
          runAction('index deleted', () =>
            overviewDeleteIndices<true>({ path: { ...clusterPath(connection), indices: index.name }, throwOnError: true }),
          ),
      ),
    flushIndex: (index: OverviewIndex) =>
      void runAction('index flushed', () =>
        overviewFlushIndices<true>({ path: { ...clusterPath(connection), indices: index.name }, throwOnError: true }),
      ),
    forceMerge: (index: OverviewIndex) =>
      void runAction('force merge started', () =>
        overviewForceMerge<true>({ path: { ...clusterPath(connection), indices: index.name }, throwOnError: true }),
      ),
    openIndex: (index: OverviewIndex) =>
      void runAction('index opened', () =>
        overviewOpenIndices<true>({ path: { ...clusterPath(connection), indices: index.name }, throwOnError: true }),
      ),
    refreshIndex: (index: OverviewIndex) =>
      void runAction('index refreshed', () =>
        overviewRefreshIndices<true>({ path: { ...clusterPath(connection), indices: index.name }, throwOnError: true }),
      ),
    showMappings: (index: OverviewIndex) =>
      void showJson(`${index.name} mappings`, () =>
        commonsGetIndexMapping<true>({ path: { ...clusterPath(connection), index: index.name }, throwOnError: true }),
      ),
    showSettings: (index: OverviewIndex) =>
      void showJson(`${index.name} settings`, () =>
        commonsGetIndexSettings<true>({ path: { ...clusterPath(connection), index: index.name }, throwOnError: true }),
      ),
    showStats: (index: OverviewIndex) =>
      void showJson(`${index.name} stats`, () =>
        commonsGetIndexStats<true>({ path: { ...clusterPath(connection), index: index.name }, throwOnError: true }),
      ),
  };

  function canReceiveShard(index: OverviewIndex | null, node: OverviewNode, shard = relocatingShard) {
    if (!index || !shard) return false;
    if (shard.index !== index.name || shard.node === node.id) return false;
    const nodeShards = index.shards?.[node.id];
    const shards = Array.isArray(nodeShards) ? nodeShards : [];
    return !shards.some((raw) => numberValue((raw as { shard?: unknown }).shard) === shard.shard);
  }

  function canRelocateShard(shard: ShardRef) {
    const index = pageElements.find((item) => item.name === shard.index) ?? null;
    return nodesList.some((node) => canReceiveShard(index, node, shard));
  }

  function relocationDisabledReason(shard: ShardRef) {
    const index = pageElements.find((item) => item.name === shard.index) ?? null;
    if (!index) return 'index is not visible on this page';
    if (nodesList.length < 2) return 'relocation requires at least two nodes';
    if (canRelocateShard(shard)) return undefined;
    return `all visible nodes already contain shard ${shard.shard} for this index`;
  }

  const shardActions = {
    canRelocate: canRelocateShard,
    relocationDisabledReason,
    select: (shard?: ShardRef) => setRelocatingShard(shard ?? null),
    selected: relocatingShard,
    showStats: (shard: ShardRef) =>
      void showJson(`${shard.index} shard ${shard.shard} stats`, () =>
        overviewShardStats<true>({
          path: { ...clusterPath(connection), index: shard.index, shard: shard.shard },
          query: { node: shard.node },
          throwOnError: true,
        }),
      ),
  };

  async function relocateShard(to: string) {
    if (!relocatingShard) return;
    await runAction('relocation started', () =>
      overviewRelocateShard<true>({
        body: {
          from: relocatingShard.node,
          to,
        },
        path: { ...clusterPath(connection), index: relocatingShard.index, shard: relocatingShard.shard },
        throwOnError: true,
      }),
    );
    setRelocatingShard(null);
  }

  if (!data) return <Loading label="loading overview" />;

  const indices = (data.indices ?? [])
    .filter((index) => (showClosed || !index.closed) && (showSpecial || !index.special))
    .filter((index) => !showOnlyAffected || index.unhealthy || Boolean(index.shards?.unassigned?.length))
    .filter((index) => `${index.name} ${(index.aliases ?? []).join(' ')}`.toLowerCase().includes(filterName.toLowerCase()))
    .sort((left, right) => (asc ? left.name.localeCompare(right.name) : right.name.localeCompare(left.name)));
  const nodesList = (data.nodes ?? []).filter((node) =>
    textValue(node.name).toLowerCase().includes(nodeFilter.toLowerCase()),
  );
  const pageCount = Math.max(1, Math.ceil(indices.length / pageSize));
  const currentPage = Math.min(page, pageCount - 1);
  const pageElements = indices.slice(currentPage * pageSize, currentPage * pageSize + pageSize);
  const pageSlots = Array.from({ length: pageSize }, (_, index) => pageElements[index] ?? null);
  const selectedIndices = pageElements.map((index) => index.name).join(',');
  const hasShardIssues =
    numberValue(data.unassigned_shards) > 0 ||
    numberValue(data.relocating_shards) > 0 ||
    numberValue(data.initializing_shards) > 0;
  const relocationTargets = relocatingShard
    ? pageSlots.some((index) => index && nodesList.some((node) => canReceiveShard(index, node)))
    : false;

  return (
    <div>
      {jsonDialog ? <JsonModal dialog={jsonDialog} onClose={() => setJsonDialog(null)} /> : null}
      {confirmDialog ? (
        <ConfirmModal
          body={confirmDialog.body}
          confirmLabel={confirmDialog.confirmLabel}
          onClose={() => setConfirmDialog(null)}
          onConfirm={confirmDialog.onConfirm}
          title={confirmDialog.title}
        />
      ) : null}
      <Stats data={data} />
      {relocatingShard && !relocationTargets ? (
        <div className="alert alert-info flex items-center justify-between">
          <span>
            shard {relocatingShard.index}/{relocatingShard.shard} selected for relocation, but there is no eligible target slot on this page
          </span>
          <button className="btn btn-default btn-xs" type="button" onClick={() => setRelocatingShard(null)}>
            cancel
          </button>
        </div>
      ) : null}
      <div className="mb-[15px] flex flex-wrap items-start gap-x-[18px] gap-y-[8px]">
        <div className="form-group m-0 w-[220px] max-w-full">
          <input
            className="form-control form-control-sm"
            placeholder="filter indices"
            type="text"
            value={filterName}
            onChange={(event) => setFilterName(event.target.value)}
          />
        </div>
        <div className="form-group m-0 pt-[7px]">
          <Checkbox checked={showClosed} className="whitespace-nowrap" label={`closed (${data.closed_indices})`} onChange={setShowClosed} />
        </div>
        <div className="form-group m-0 pt-[7px]">
          <Checkbox checked={showSpecial} className="whitespace-nowrap" label={`.special (${data.special_indices})`} onChange={setShowSpecial} />
        </div>
        <div className="form-group m-0 w-[190px] max-w-full">
          <input
            className="form-control form-control-sm"
            placeholder="filter nodes"
            type="text"
            value={nodeFilter}
            onChange={(event) => setNodeFilter(event.target.value)}
          />
        </div>
        <div className="form-group m-0 ml-auto max-w-full">
          <Pagination page={currentPage} pageSize={pageSize} setPage={setPage} total={indices.length} />
        </div>
      </div>
      <table className="table table-bordered table-condensed table-rounded table-inverse shard-map">
        <thead>
          <tr>
            <td>
              <div className="grid grid-cols-4 items-start">
                <div className="min-w-0">
                  {data.shard_allocation ? (
                    <div className="group relative">
                      <span className="title normal-action" title="disable shard allocation">
                        <Icon className="table-control" name="unlock" size={28} />
                      </span>
                      <ul className="absolute top-full left-0 z-[1000] hidden min-w-[160px] list-none border border-[#55595c] bg-[#373a3c] py-[5px] text-left shadow-lg group-hover:block group-focus-within:block [&>li>a]:block [&>li>a]:whitespace-nowrap [&>li>a]:px-5 [&>li>a]:py-[3px] [&>li>a:hover]:bg-[#434749] [&>li>a:hover]:text-white">
                        {['none', 'primaries', 'new_primaries'].map((kind) => (
                          <li key={kind}>
                            <a
                              target="_self"
                              onClick={() =>
                                void runAction('shard allocation disabled', () =>
                                  overviewDisableShardAllocation<true>({
                                    body: { kind },
                                    path: clusterPath(connection),
                                    throwOnError: true,
                                  }),
                                )
                              }
                            >
                              <Icon name="lock" /> {kind}
                              {kind === 'none' ? ' (default)' : ''}
                            </a>
                          </li>
                        ))}
                      </ul>
                    </div>
                  ) : (
                    <Icon
                      className="table-control red normal-action"
                      name="lock"
                      size={28}
                      title="enable shard allocation"
                      onClick={() =>
                        void runAction('shard allocation enabled', () =>
                          overviewEnableShardAllocation<true>({ path: clusterPath(connection), throwOnError: true }),
                        )
                      }
                    />
                  )}
                </div>
                <div className="min-w-0">
                  <Icon
                    className="normal-action table-control"
                    name={expandedView ? 'compress' : 'expand'}
                    size={28}
                    title={expandedView ? 'condense view' : 'expand view'}
                    onClick={() => setExpandedView((value) => !value)}
                  />
                </div>
                <div className="min-w-0">
                  <Icon
                    className="normal-action table-control"
                    name={asc ? 'sort-alpha-asc' : 'sort-alpha-desc'}
                    size={28}
                    title={asc ? 'sort ascending' : 'sort descending'}
                    onClick={() => setAsc((value) => !value)}
                  />
                </div>
                <div className="group relative min-w-0">
                  <span className="title normal-action" title="more options">
                    <Icon className="table-control" name="caret-down" size={28} />
                  </span>
                  <ul className="absolute top-full left-0 z-[1000] hidden min-w-[160px] list-none border border-[#55595c] bg-[#373a3c] py-[5px] text-left shadow-lg group-hover:block group-focus-within:block [&>li>a]:block [&>li>a]:whitespace-nowrap [&>li>a]:px-5 [&>li>a]:py-[3px] [&>li>a:hover]:bg-[#434749] [&>li>a:hover]:text-white">
                    <li>
                      <a onClick={() => void runAction('selected indices closed', () => overviewCloseIndices<true>({ path: { ...clusterPath(connection), indices: selectedIndices }, throwOnError: true }))}>
                        <Icon name="folder" /> close selected
                      </a>
                    </li>
                    <li>
                      <a onClick={() => void runAction('selected indices opened', () => overviewOpenIndices<true>({ path: { ...clusterPath(connection), indices: selectedIndices }, throwOnError: true }))}>
                        <Icon name="folder-open" /> open selected
                      </a>
                    </li>
                    <li>
                      <a onClick={() => void runAction('force merge started', () => overviewForceMerge<true>({ path: { ...clusterPath(connection), indices: selectedIndices }, throwOnError: true }))}>
                        <Icon name="wrench" /> force merge selected
                      </a>
                    </li>
                    <li>
                      <a onClick={() => void runAction('selected indices refreshed', () => overviewRefreshIndices<true>({ path: { ...clusterPath(connection), indices: selectedIndices }, throwOnError: true }))}>
                        <Icon name="refresh" /> refresh selected
                      </a>
                    </li>
                    <li>
                      <a onClick={() => void runAction('selected indices flushed', () => overviewFlushIndices<true>({ path: { ...clusterPath(connection), indices: selectedIndices }, throwOnError: true }))}>
                        <Icon name="gavel" /> flush selected
                      </a>
                    </li>
                    <li>
                      <a onClick={() => void runAction('selected caches cleared', () => overviewClearIndicesCache<true>({ path: { ...clusterPath(connection), indices: selectedIndices }, throwOnError: true }))}>
                        <Icon name="circle" /> clear selected caches
                      </a>
                    </li>
                    <li className="divider" role="separator" />
                    <li>
                      <a
                        onClick={() =>
                          confirmDelete(
                            `Delete ${pageElements.length} selected indices?`,
                            `Delete selected indices: ${selectedIndices}? This operation cannot be undone.`,
                            () =>
                              runAction('selected indices deleted', () =>
                                overviewDeleteIndices<true>({
                                  path: { ...clusterPath(connection), indices: selectedIndices },
                                  throwOnError: true,
                                }),
                              ),
                          )
                        }
                      >
                        <Icon className="red" name="trash" /> delete selected
                      </a>
                    </li>
                  </ul>
                </div>
              </div>
            </td>
            {pageSlots.map((index, slot) => (
              <td className={index?.closed ? 'closed-index' : ''} key={index?.name ?? `empty-${slot}`}>
                {index ? (
                  <IndexHeader
                    actions={indexActions}
                    dataExplorerHref={
                      dataExplorerEnabled
                        ? `#/data_explorer?host=${encodeURIComponent(connection.host)}&index=${encodeURIComponent(index.name)}`
                        : undefined
                    }
                    index={index}
                    settingsHref={`#/index_settings?host=${encodeURIComponent(connection.host)}&index=${encodeURIComponent(index.name)}`}
                  />
                ) : null}
              </td>
            ))}
          </tr>
        </thead>
        <tbody>
          {hasShardIssues || showOnlyAffected ? (
            <tr>
              <td>
                {numberValue(data.unassigned_shards) > 0 ? (
                  <div className="subtitle">
                    <Icon className="alert-warning" name="warning" /> {formatNumber(data.unassigned_shards)} unassigned shards
                  </div>
                ) : null}
                {numberValue(data.relocating_shards) > 0 ? (
                  <div className="subtitle">
                    <Icon name="refresh" spin /> {formatNumber(data.relocating_shards)} relocating shards
                  </div>
                ) : null}
                {numberValue(data.initializing_shards) > 0 ? (
                  <div className="subtitle">
                    <Icon name="spinner" spin /> {formatNumber(data.initializing_shards)} initializing shards
                  </div>
                ) : null}
                {!hasShardIssues && showOnlyAffected ? (
                  <div className="subtitle">
                    <Icon name="check" /> no affected indices
                  </div>
                ) : null}
                <div>
                  {!showOnlyAffected ? (
                    <span className="normal-action" onClick={() => setShowOnlyAffected(true)}>
                      <i>
                        <small>show only affected indices</small>
                      </i>
                    </span>
                  ) : (
                    <span className="normal-action" onClick={() => setShowOnlyAffected(false)}>
                      <i>
                        <small>show all indices</small>
                      </i>
                    </span>
                  )}
                </div>
              </td>
              {pageSlots.map((index, slot) => (
                <td key={index?.name ?? `empty-unassigned-${slot}`}>{index ? renderShards(index.shards?.unassigned) : null}</td>
              ))}
            </tr>
          ) : null}
          {nodesList.sort(byName).map((node) => (
            <tr key={node.id}>
              <td>
                <OverviewNodeCell expanded={expandedView} node={node} />
              </td>
              {pageSlots.map((index, slot) => (
                <td key={index?.name ?? `empty-${node.id}-${slot}`}>
                  {index ? renderShards(index.shards?.[node.id], index.closed, shardActions) : null}
                  {canReceiveShard(index, node) ? (
                    <span
                      className="shard shard-spot normal-action"
                      title={`relocate shard to ${node.name}`}
                      onClick={() => void relocateShard(node.id)}
                    >
                      <Icon name="download" />
                    </span>
                  ) : null}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function JsonModal({ dialog, onClose }: { dialog: JsonDialog; onClose: () => void }) {
  useEscape(onClose);

  return (
    <ModalFrame onClose={onClose} title={dialog.title}>
      <div className="modal-body">
        <LazyJsonEditor height={520} readOnly value={formatJson(dialog.body)} onChange={() => {}} />
      </div>
      <div className="modal-footer">
        <button className="btn btn-default" type="button" onClick={onClose}>
          close
        </button>
      </div>
    </ModalFrame>
  );
}

function getOverviewPageSize(): number {
  return Math.max(Math.round(window.innerWidth / 280), 1);
}
