import { Fragment, type MouseEvent } from 'react';

import type { Node, Overview, OverviewIndex, OverviewNode } from '../api/client';
import { formatBytes, formatNumber, numberValue, textValue } from '../utils/format';
import { Icon, type IconName } from './Icon';

const titleClass = 'block overflow-hidden text-ellipsis whitespace-nowrap';
const nodeBadgesClass = 'float-left inline-block w-5 overflow-hidden text-ellipsis whitespace-nowrap';
const labelPrimaryClass = 'inline bg-[#1ca8dd] px-[.6em] py-[.2em] text-[75%] font-bold leading-none text-[#373a3c]';
const labelSuccessClass = 'inline bg-[#1ac98e] px-[.6em] py-[.2em] text-[75%] font-bold leading-none text-[#373a3c]';

export function Loading({ label }: { label: string }) {
  return (
    <div className="text-center">
      <Icon name="spinner" spin /> {label}
    </div>
  );
}

export function Pagination({
  page,
  pageSize,
  setPage,
  total,
}: {
  page: number;
  pageSize: number;
  setPage: (page: number) => void;
  total: number;
}) {
  const first = total > 0 ? page * pageSize + 1 : 0;
  const last = Math.min(total, (page + 1) * pageSize);
  const previous = page > 0;
  const next = (page + 1) * pageSize < total;

  return (
    <div className="row">
      <div className="col-lg-12">
        <nav className="pull-right">
          <ul className="pager noselect">
            <li>
              <span className="pager-text">
                {formatNumber(first)}-{formatNumber(last)} of {formatNumber(total)}
              </span>
            </li>
            <li className={previous ? 'normal-action' : 'disabled'} onClick={() => previous && setPage(page - 1)}>
              <a target="_self">&larr;</a>
            </li>
            <li className={next ? 'normal-action' : 'disabled'} onClick={() => next && setPage(page + 1)}>
              <a target="_self">&rarr;</a>
            </li>
          </ul>
        </nav>
      </div>
    </div>
  );
}

export function SortLink({
  active,
  onClick,
  reverse,
  text,
}: {
  active: boolean;
  onClick: () => void;
  reverse: boolean;
  text: string;
}) {
  return (
    <span className="normal-action" onClick={onClick}>
      {text} {active ? <Icon name={reverse ? 'sort-alpha-desc' : 'sort-alpha-asc'} /> : null}
    </span>
  );
}

export function Stats({ data }: { data: Overview }) {
  return (
    <div className="row">
      <div className="col-xs-12">
        <div className="stats">
          <div className="row">
            <Stat value={textValue(data.cluster_name)} />
            <Stat label="nodes" value={formatNumber(data.number_of_nodes)} />
            <Stat label="indices" value={formatNumber(data.indices?.length ?? 0)} />
            <Stat
              label="shards"
              value={formatNumber(
                numberValue(data.active_shards) +
                  numberValue(data.initializing_shards) +
                  numberValue(data.relocating_shards) +
                  numberValue(data.unassigned_shards),
              )}
            />
            <Stat label="docs" value={formatNumber(data.docs_count)} />
            <Stat value={formatBytes(data.size_in_bytes)} />
          </div>
        </div>
      </div>
    </div>
  );
}

function Stat({ label, value }: { label?: string; value: string }) {
  return (
    <div className="col-lg-2">
      <span className="stat">
        <span className="stat-value">{value}</span>
        {label ? <span> {label}</span> : null}
      </span>
    </div>
  );
}

export type IndexHeaderActions = {
  clearCache: (index: OverviewIndex) => void;
  closeIndex: (index: OverviewIndex) => void;
  deleteIndex: (index: OverviewIndex) => void;
  flushIndex: (index: OverviewIndex) => void;
  forceMerge: (index: OverviewIndex) => void;
  openIndex: (index: OverviewIndex) => void;
  refreshIndex: (index: OverviewIndex) => void;
  showMappings: (index: OverviewIndex) => void;
  showSettings: (index: OverviewIndex) => void;
  showStats: (index: OverviewIndex) => void;
};

export type ShardRef = {
  index: string;
  node: string;
  shard: number;
};

export type ShardActions = {
  canRelocate?: (shard: ShardRef) => boolean;
  relocationDisabledReason?: (shard: ShardRef) => string | undefined;
  select: (shard?: ShardRef) => void;
  selected?: ShardRef | null;
  showStats: (shard: ShardRef) => void;
};

export function IndexHeader({
  actions,
  dataExplorerHref,
  index,
  settingsHref,
}: {
  actions: IndexHeaderActions;
  dataExplorerHref?: string;
  index: OverviewIndex;
  settingsHref: string;
}) {
  function action(handler: (index: OverviewIndex) => void) {
    return (event: MouseEvent<HTMLAnchorElement>) => {
      event.preventDefault();
      handler(index);
    };
  }

  return (
    <div>
      <div className="group relative">
        <span className="title normal-action" title={index.name}>
          <Icon className="pull-right" name="caret-down" />
          {index.name}
        </span>
        <ul className="absolute top-full right-0 z-[1000] hidden min-w-[160px] list-none border border-[#55595c] bg-[#373a3c] py-[5px] text-left shadow-lg group-hover:block group-focus-within:block [&>li>a]:block [&>li>a]:whitespace-nowrap [&>li>a]:px-5 [&>li>a]:py-[3px] [&>li>a:hover]:bg-[#434749] [&>li>a:hover]:text-white">
          {dataExplorerHref && !index.closed ? (
            <li>
              <a href={dataExplorerHref} target="_self">
                <Icon name="database" /> browse data
              </a>
            </li>
          ) : null}
          <li>
            <a target="_self" onClick={action(actions.showSettings)}>
              <Icon name="info" /> show settings
            </a>
          </li>
          <li>
            <a target="_self" onClick={action(actions.showMappings)}>
              <Icon name="code" /> show mappings
            </a>
          </li>
          <li>
            <a target="_self" onClick={action(actions.showStats)}>
              <Icon name="info" /> show stats
            </a>
          </li>
          {index.closed ? (
            <li>
              <a target="_self" onClick={action(actions.openIndex)}>
                <Icon name="folder-open" /> open index
              </a>
            </li>
          ) : (
            <li>
              <a target="_self" onClick={action(actions.closeIndex)}>
                <Icon name="folder" /> close index
              </a>
            </li>
          )}
          <li>
            <a target="_self" onClick={action(actions.forceMerge)}>
              <Icon name="wrench" /> force merge
            </a>
          </li>
          <li>
            <a target="_self" onClick={action(actions.refreshIndex)}>
              <Icon name="refresh" /> refresh index
            </a>
          </li>
          <li>
            <a target="_self" onClick={action(actions.flushIndex)}>
              <Icon name="gavel" /> flush index
            </a>
          </li>
          <li>
            <a target="_self" onClick={action(actions.clearCache)}>
              <Icon name="circle" /> clear cache
            </a>
          </li>
          <li>
            <a href={settingsHref} target="_self">
              <Icon name="cog" /> index settings
            </a>
          </li>
          <li className="divider" />
          <li>
            <a target="_self" onClick={action(actions.deleteIndex)}>
              <Icon className="red" name="trash" /> delete index
            </a>
          </li>
        </ul>
      </div>
      {index.aliases?.length ? (
        <div className="subtitle">
          <div className="title">
            <Icon name="tag" /> {index.aliases[0]}
            {index.aliases.length > 1 ? <span>(+{index.aliases.length - 1})</span> : null}
          </div>
        </div>
      ) : null}
      <div className="detail">
        {!index.closed ? (
          <span>
            <span>
              <small>
                shards: {index.num_shards} * {index.num_replicas + 1} |
              </small>
            </span>{' '}
            <span>
              <small>docs: {formatNumber(index.doc_count)} |</small>
            </span>{' '}
            <span>
              <small>size: {formatBytes(index.size_in_bytes)}</small>
            </span>
          </span>
        ) : (
          <span>
            <small>
              <i>index closed</i>
            </small>
          </span>
        )}
      </div>
    </div>
  );
}

export function OverviewNodeCell({ expanded, node }: { expanded: boolean; node: OverviewNode }) {
  return (
    <>
      <div className="row">
        <div className="col-lg-12">
          <NodeBadges node={node} />
          <div className="node-info">
            <div className="title">
              <span className="normal-action">{textValue(node.name)}</span>
            </div>
            <div>
              <small>{textValue(node.host)}</small>
            </div>
          </div>
        </div>
      </div>
      <div className="row">
        <div className="col-xs-12 node-attrs">{renderAttributes(node.attributes)}</div>
      </div>
      {expanded ? (
        <div className="node-labels">
          <span className="label label-primary">JVM: {textValue(node.jvm_version)}</span>{' '}
          <span className="label label-primary">ES: {textValue(node.es_version)}</span>
        </div>
      ) : null}
      <div className="row row-condensed">
        <Progress value={node.heap?.used_percent} text="heap" />
        <Progress value={node.disk?.used_percent} text="disk" />
        <Progress value={node.cpu_percent} text="cpu" />
        <Progress max={node.available_processors} value={node.load_average} text="load" />
      </div>
    </>
  );
}

export function NodeCell({ node }: { node: Node }) {
  return (
    <>
      <div>
        <div>
          <NodeBadges node={node} />
          <div className="ml-5">
            <div className={titleClass}>
              <span>{textValue(node.name)}</span>
            </div>
            <div>
              <small>{textValue(node.host)}</small>
            </div>
          </div>
        </div>
      </div>
      <div>
        <div className="overflow-hidden text-ellipsis">{renderAttributes(node.attributes)}</div>
      </div>
      <div className="pt-[5px]">
        {node.jvm ? <span className={labelPrimaryClass}>JVM: {textValue(node.jvm)}</span> : null}{' '}
        <span className={labelPrimaryClass}>ES: {textValue(node.version)}</span>
      </div>
    </>
  );
}

function NodeBadges({
  node,
}: {
  node: Pick<Node | OverviewNode, 'current_master' | 'data' | 'ingest' | 'master'> & { coordinating?: boolean };
}) {
  return (
    <div className={nodeBadgesClass}>
      {node.master ? (
        <div>
          <Icon name={node.current_master ? 'star' : 'star-o'} title={node.current_master ? 'current master' : 'master eligible'} />
        </div>
      ) : null}
      {node.data ? (
        <div>
          <Icon name="hdd" title="data node" />
        </div>
      ) : null}
      {node.coordinating ? (
        <div>
          <Icon name="crosshairs" title="coordinating node" />
        </div>
      ) : null}
      {node.ingest ? (
        <div>
          <Icon name="crop" title="ingest node" />
        </div>
      ) : null}
    </div>
  );
}

function Progress({ max = 100, text, value }: { max?: unknown; text: string; value: unknown }) {
  const percent = Math.min(100, Math.max(0, (numberValue(value) / Math.max(1, numberValue(max))) * 100));
  const display = text === 'load' ? numberValue(value).toFixed(2) : formatNumber(value);
  return (
    <div className="col-lg-3 col-condensed">
      <div title={`${text}: ${numberValue(value).toFixed(2)}`}>
        <span className="detail">
          <small>{text}</small>
        </span>
        <div className="progress progress-thin">
          <div className={percent > 75 ? 'progress-bar-danger' : 'progress-bar-info'} style={{ width: `${percent}%` }}>
            {display}%
          </div>
        </div>
      </div>
    </div>
  );
}

export function renderAttributes(attributes?: Record<string, unknown>) {
  return Object.entries(attributes ?? {}).map(([attr, value]) => (
    <Fragment key={attr}>
      <span className={labelSuccessClass} title={attr}>
        {textValue(value)}
      </span>{' '}
    </Fragment>
  ));
}

export function renderShards(input: unknown, closed = false, actions?: ShardActions) {
  const shards = (Array.isArray(input) ? [...input] : []).sort(compareShardEntries);
  return shards.map((raw, index) => {
    const shard = raw as { node?: unknown; primary?: unknown; shard?: unknown; state?: unknown };
    const state = textValue(shard.state).toLowerCase();
    const replica = !Boolean(shard.primary) && Boolean(shard.node);
    const shardRef = {
      index: textValue((raw as { index?: unknown }).index),
      node: textValue(shard.node),
      shard: numberValue(shard.shard),
    };
    const selected =
      actions?.selected &&
      actions.selected.index === shardRef.index &&
      actions.selected.node === shardRef.node &&
      actions.selected.shard === shardRef.shard;
    const relocationDisabledReason = actions?.relocationDisabledReason?.(shardRef);

    return (
      <span className="group relative inline-block mr-[4px] mb-[4px]" key={index}>
        <span
          className={`shard shard-${state || 'unassigned'} ${replica ? 'shard-replica' : ''} ${
            closed ? 'shard-closed' : 'normal-action'
          } ${selected ? '!border-[#e4d836] !bg-[#2f3023] !text-[#e4d836]' : ''}`}
        >
          <small>{textValue(shard.shard)}</small>
        </span>
        {!closed ? (
          <ul className="absolute top-full left-0 z-[1000] hidden min-w-[160px] list-none border border-[#55595c] bg-[#373a3c] py-[5px] text-left shadow-lg group-hover:block group-focus-within:block [&>li>a]:block [&>li>a]:whitespace-nowrap [&>li>a]:px-5 [&>li>a]:py-[3px] [&>li>a:hover]:bg-[#434749] [&>li>a:hover]:text-white">
            <li>
              <a
                target="_self"
                onClick={(event) => {
                  event.preventDefault();
                  actions?.showStats(shardRef);
                }}
              >
                <Icon name="info" /> display shard stats
              </a>
            </li>
            <li>
              <a
                className={relocationDisabledReason ? 'cursor-not-allowed opacity-40' : ''}
                target="_self"
                title={relocationDisabledReason}
                onClick={(event) => {
                  event.preventDefault();
                  if (!actions || relocationDisabledReason) return;
                  actions.select(selected ? undefined : shardRef);
                }}
              >
                <Icon name="arrows" /> {selected ? 'unselect for relocation' : 'select for relocation'}
              </a>
            </li>
          </ul>
        ) : null}
      </span>
    );
  });
}

function compareShardEntries(left: unknown, right: unknown) {
  const leftShard = left as { primary?: unknown; shard?: unknown };
  const rightShard = right as { primary?: unknown; shard?: unknown };
  const shardDiff = numberValue(leftShard.shard) - numberValue(rightShard.shard);
  if (shardDiff !== 0) return shardDiff;
  if (Boolean(leftShard.primary) === Boolean(rightShard.primary)) return 0;
  return Boolean(leftShard.primary) ? -1 : 1;
}

export function byName(left: OverviewNode, right: OverviewNode): number {
  return textValue(left.name).localeCompare(textValue(right.name));
}

export function compareNodes(
  left: Node,
  right: Node,
  sortBy: keyof Node | 'cpu.load' | 'cpu.process' | 'heap.percent' | 'disk.percent',
  reverse: boolean,
): number {
  const leftValue = pathValue(left, sortBy);
  const rightValue = pathValue(right, sortBy);
  const compared =
    typeof leftValue === 'number' || typeof rightValue === 'number'
      ? numberValue(leftValue) - numberValue(rightValue)
      : textValue(leftValue).localeCompare(textValue(rightValue));
  return reverse ? -compared : compared;
}

function pathValue(
  node: Node,
  path: keyof Node | 'cpu.load' | 'cpu.process' | 'heap.percent' | 'disk.percent',
): unknown {
  if (path === 'cpu.load') return node.cpu?.load;
  if (path === 'cpu.process') return node.cpu?.process;
  if (path === 'heap.percent') return node.heap?.percent;
  if (path === 'disk.percent') return node.disk?.percent;
  return node[path];
}

export function roleIcon(role: string): IconName {
  if (role === 'master') return 'star';
  if (role === 'data') return 'hdd';
  if (role === 'ingest') return 'crop';
  return 'crosshairs';
}
