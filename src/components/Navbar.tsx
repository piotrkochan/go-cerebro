import { Link } from '@tanstack/react-router';
import { useEffect, useState } from 'react';

import { indexSettingsUpdate, type HostBodyWritable } from '../api/client';
import { alertsActions } from '../stores/alertsStore';
import { CerebroLogo } from './CerebroLogo';
import { DataTable, type DataTableColumn } from './DataTable';
import { Icon } from './Icon';
import { ConfirmModal, ModalFrame } from './Modal';
import type { ClusterHealthFix, ClusterHealthIssue } from '../stores/sessionStore';
import { clusterPath } from '../utils/connection';
import { errorMessage, timeInterval } from '../utils/format';

const refreshOptions = [5000, 10000, 15000, 30000, 60000];

export function Navbar({
  connected,
  connection,
  disconnect,
  healthIssue,
  host,
  onHealthFixed,
  refreshInterval,
  setRefreshInterval,
  status,
}: {
  connected: boolean;
  connection: HostBodyWritable;
  disconnect: () => void;
  healthIssue: ClusterHealthIssue | null;
  host: string;
  onHealthFixed: () => void;
  refreshInterval: number;
  setRefreshInterval: (value: number) => void;
  status: string;
}) {
  const [healthOpen, setHealthOpen] = useState(false);
  const [fixConfirm, setFixConfirm] = useState<ClusterHealthFix | null>(null);
  useEffect(() => {
    setHealthOpen(false);
    setFixConfirm(null);
  }, [host, status]);

  async function applyFix(fix: ClusterHealthFix) {
    if (fix.action !== 'set_index_replicas') return;
    try {
      await indexSettingsUpdate<true>({
        body: { [fix.setting]: fix.value },
        path: { ...clusterPath(connection), index: fix.index },
        throwOnError: true,
      });
      alertsActions.notify('info', `Applied fix: ${fix.summary}`);
      onHealthFixed();
    } catch (error) {
      alertsActions.notify('danger', `Error applying fix: ${errorMessage(error)}`);
    }
  }

  if (!connected) return null;
  const search = { host: connection.host };
  const statusValue = status.toLowerCase();
  const showHealthIssue = statusValue === 'yellow' || statusValue === 'red';
  const statusBorder =
    statusValue === 'green'
      ? 'border-[#1AC98E]'
      : statusValue === 'yellow'
        ? 'border-[#E4D836]'
        : statusValue === 'red'
          ? 'border-[#E64759]'
          : 'border-[#55595c]';
  const navLink = 'flex h-[50px] items-center gap-1.5 px-[15px] text-[#eceeef] hover:bg-[#434749] hover:text-white';
  const menu =
    'absolute top-full left-0 z-[1000] hidden min-w-[160px] list-none border border-[#55595c] bg-[#373a3c] py-[5px] text-left shadow-lg group-hover:block group-focus-within:block [&>li>a]:flex [&>li>a]:items-center [&>li>a]:gap-1.5 [&>li>a]:whitespace-nowrap [&>li>a]:px-5 [&>li>a]:py-[3px] [&>li>a:hover]:bg-[#434749] [&>li>a:hover]:text-white';
  const healthColor = statusValue === 'red' ? 'text-[#E64759]' : 'text-[#E4D836]';
  const healthFill = statusValue === 'red' ? 'fill-[#E64759]' : 'fill-[#E4D836]';

  return (
    <nav className={`fixed top-0 right-0 left-0 z-[1030] min-h-[50px] border-b-[5px] bg-[#373a3c] ${statusBorder}`}>
      {fixConfirm ? (
        <ConfirmModal
          body={
            <div>
              <p>
                Change <strong>{fixConfirm.setting}</strong> on index <strong>{fixConfirm.index}</strong> to <strong>{fixConfirm.value}</strong>?
              </p>
              <p className="info-text">{fixConfirm.rationale}</p>
              <pre className="mt-3 border border-[#55595c] bg-[#2f3234] p-3 text-[#eceeef]">
                {JSON.stringify({ [fixConfirm.setting]: fixConfirm.value }, null, 2)}
              </pre>
            </div>
          }
          confirmClassName="btn-warning"
          confirmLabel={
            <>
              <Icon name="wrench" /> apply fix
            </>
          }
          title="fix cluster health"
          onClose={() => setFixConfirm(null)}
          onConfirm={() => applyFix(fixConfirm)}
        />
      ) : null}
      <div className="mx-auto flex min-h-[50px] w-full items-stretch px-[15px]">
        <div className="flex items-stretch">
          <button className="hidden" type="button">
            <span className="sr-only">Toggle navigation</span>
            <span className="icon-bar" />
            <span className="icon-bar" />
            <span className="icon-bar" />
          </button>
          <Link className="flex h-[50px] items-center px-[15px]" search={search} to="/overview">
            <CerebroLogo size="header" />
          </Link>
        </div>
        <div id="navbar" className="flex flex-1 items-stretch justify-between">
          <ul className="m-0 flex list-none p-0">
            <li>
              <Link className={navLink} search={search} to="/overview">
                <Icon name="sitemap" /> overview
              </Link>
            </li>
            <li>
              <Link className={navLink} search={search} to="/nodes">
                <Icon name="server" /> nodes
              </Link>
            </li>
            <li>
              <Link className={navLink} search={search} to="/rest">
                <Icon name="edit" /> rest
              </Link>
            </li>
            <li className="group relative">
              <a className={navLink} href="#more">
                <Icon name="magic" /> more <Icon name="caret-down" />
              </a>
              <ul className={menu}>
                <li>
                  <Link search={search} to="/create">
                    <Icon name="file" /> create index
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/cluster_settings">
                    <Icon name="cogs" /> cluster settings
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/aliases">
                    <Icon name="tags" /> aliases
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/analysis">
                    <Icon name="puzzle" /> analysis
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/templates">
                    <Icon name="book" /> index templates
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/data_streams">
                    <Icon name="database" /> data streams
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/ilm">
                    <Icon name="history" /> ilm policies
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/repositories">
                    <Icon name="database" /> repositories
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/snapshot">
                    <Icon name="camera" /> snapshot
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/cat">
                    <Icon name="list" /> cat apis
                  </Link>
                </li>
              </ul>
            </li>
          </ul>
          <ul className="m-0 ml-auto flex list-none p-0">
            <li className="group relative">
              <a className={navLink} href="#refresh">
                <Icon name="refresh" /> {timeInterval(refreshInterval)} <Icon name="caret-down" />
              </a>
              <ul className={`${menu} min-w-[60px]`}>
                {refreshOptions.map((value) => (
                  <li key={value}>
                    <a className="cursor-pointer" onClick={() => setRefreshInterval(value)}>
                      {timeInterval(value)}
                    </a>
                  </li>
                ))}
              </ul>
            </li>
            {showHealthIssue ? (
              <li className="relative">
                <button className={`${navLink} ${healthColor} cursor-pointer border-0 bg-transparent`} type="button" onClick={() => setHealthOpen((open) => !open)}>
                  <Icon className={healthFill} name="circle" size={9} /> {statusValue}
                </button>
                {healthOpen ? <HealthIssuePanel issue={healthIssue} status={statusValue} onFix={setFixConfirm} /> : null}
              </li>
            ) : null}
            <li className="hidden sm:block">
              <a className={navLink}>{host}</a>
            </li>
            <li>
              <a className={`${navLink} hidden cursor-pointer sm:flex`} onClick={disconnect}>
                <Icon name="plug" />
              </a>
            </li>
          </ul>
        </div>
      </div>
    </nav>
  );
}

function HealthIssuePanel({
  issue,
  onFix,
  status,
}: {
  issue: ClusterHealthIssue | null;
  onFix: (fix: ClusterHealthFix) => void;
  status: string;
}) {
  const shards = issue?.unassigned_shards ?? [];
  const fixes = issue?.fixes ?? [];
  const [showAll, setShowAll] = useState(false);
  const visibleShards = shards.slice(0, 5);
  return (
    <>
      {showAll ? (
        <ModalFrame onClose={() => setShowAll(false)} title="cluster health details">
          <div className="modal-body">
            <HealthShardTable shards={shards} />
          </div>
        </ModalFrame>
      ) : null}
      <div className="absolute top-full right-0 z-[1000] w-[560px] border border-[#55595c] bg-[#373a3c] p-[12px] text-left shadow-lg">
        <div className="mb-[8px] flex items-center justify-between border-b border-[#55595c] pb-[8px]">
          <span className={status === 'red' ? 'text-[#E64759]' : 'text-[#E4D836]'}>
            <Icon className={status === 'red' ? 'fill-[#E64759]' : 'fill-[#E4D836]'} name="circle" size={9} /> cluster {status}
          </span>
          <span className="info-text">{issue?.summary ?? `cluster health is ${status}`}</span>
        </div>
        {fixes.length ? (
          <div className="mb-[10px] border border-[#55595c] bg-[#343739] p-[8px]">
            {fixes.map((fix) => (
              <div className="flex items-center justify-between gap-[10px]" key={`${fix.action}-${fix.index}-${fix.setting}`}>
                <div>
                  <div>
                    <Icon name="wrench" /> {fix.summary}
                  </div>
                  <div className="info-text">{fix.rationale}</div>
                </div>
                <button className="btn btn-warning btn-xs whitespace-nowrap" type="button" onClick={() => onFix(fix)}>
                  fix
                </button>
              </div>
            ))}
          </div>
        ) : null}
        {shards.length ? (
          <>
            <HealthShardTable shards={visibleShards} />
            {shards.length > 5 ? (
              <div className="mt-[8px] text-right">
                <button className="btn btn-default btn-xs" type="button" onClick={() => setShowAll(true)}>
                  more ({shards.length - 5})
                </button>
              </div>
            ) : null}
          </>
        ) : (
          <div className="info-text">No unassigned shard details reported by Elasticsearch.</div>
        )}
      </div>
    </>
  );
}

function HealthShardTable({ shards }: { shards: ClusterHealthIssue['unassigned_shards'] }) {
  const [expanded, setExpanded] = useState<string | null>(null);
  const columns: DataTableColumn<ClusterHealthIssue['unassigned_shards'][number]>[] = [
    { header: 'index', key: 'index', render: (shard) => shard.index },
    { header: 'shard', key: 'shard', render: (shard) => shard.shard },
    { header: 'type', key: 'type', render: (shard) => (shard.primary_replica === 'p' ? 'primary' : 'replica') },
    { header: 'reason', key: 'reason', render: (shard) => shard.reason || 'not reported' },
    { header: 'blocked by', key: 'blocked-by', render: (shard) => (shard.deciders.length ? shard.deciders.join(', ') : shard.allocation_decision || 'not reported') },
    {
      className: 'text-right',
      header: null,
      key: 'details',
      render: (shard, index) => {
        const key = healthShardKey(shard, index);
        return shard.explanation ? (
          <button className="btn btn-default btn-xs" type="button" onClick={() => setExpanded(expanded === key ? null : key)}>
            details
          </button>
        ) : null;
      },
    },
  ];

  return (
    <DataTable
      className="mb-0"
      columns={columns}
      getRowKey={healthShardKey}
      rows={shards}
      renderDetail={(shard, index) => (expanded === healthShardKey(shard, index) ? shard.explanation : null)}
    />
  );
}

function healthShardKey(shard: ClusterHealthIssue['unassigned_shards'][number], index: number) {
  return `${shard.index}-${shard.shard}-${shard.primary_replica}-${index}`;
}
