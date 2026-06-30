import { useEffect, useState } from 'react';
import type { ReactNode } from 'react';

import {
  snapshotsCreate,
  snapshotsDelete,
  snapshotsGet,
  snapshotsLoad,
  snapshotsRestore,
  type HostBodyWritable,
} from '../api/client';
import { DataTable, SortHeader, type DataTableColumn } from '../components/DataTable';
import { Icon } from '../components/Icon';
import { ConfirmModal } from '../components/Modal';
import { SplitPane } from '../components/SplitPane';
import type { Notify } from '../types';
import { clusterPath } from '../utils/connection';
import { errorMessage, textValue } from '../utils/format';
import { nextSort, sortByText, type SortState } from '../utils/sort';

type SnapshotIndex = { name: string; special?: boolean };
type Snapshot = { indices?: string[]; snapshot?: string; start_time?: unknown; state?: unknown; uuid?: string };
type SnapshotLoad = { indices?: SnapshotIndex[]; repositories?: string[] };
type SnapshotSortKey = 'name' | 'start_time' | 'state';

export function SnapshotPage({
  connection,
  notify,
  refreshTick,
}: {
  connection: HostBodyWritable;
  notify: Notify;
  refreshTick: number;
}) {
  const [indices, setIndices] = useState<SnapshotIndex[]>([]);
  const [repositories, setRepositories] = useState<string[]>([]);
  const [repository, setRepository] = useState('');
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [showSpecial, setShowSpecial] = useState(false);
  const [deleteSnapshotName, setDeleteSnapshotName] = useState('');
  const [restoreSnapshotName, setRestoreSnapshotName] = useState('');
  const [sort, setSort] = useState<SortState<SnapshotSortKey>>({ key: 'name', order: 'asc' });
  const [createForm, setCreateForm] = useState({ ignoreUnavailable: false, includeGlobalState: true, indices: [] as string[], repository: '', snapshot: '' });
  const [restoreForms, setRestoreForms] = useState<Record<string, RestoreForm>>({});

  useEffect(() => {
    async function load() {
      try {
        const result = await snapshotsGet<true>({ path: clusterPath(connection), throwOnError: true });
        const data = result.data.data as SnapshotLoad;
        setIndices(data.indices ?? []);
        setRepositories((data.repositories ?? []).sort());
      } catch (error) {
        notify('danger', `Error loading repositories: ${errorMessage(error)}`);
      }
    }
    void load();
  }, [connection, notify, refreshTick]);

  async function loadSnapshots(nextRepository: string) {
    setRepository(nextRepository);
    if (!nextRepository) return setSnapshots([]);
    try {
      const result = await snapshotsLoad<true>({ path: { ...clusterPath(connection), repository: nextRepository }, throwOnError: true });
      setSnapshots(Array.isArray(result.data.data) ? result.data.data as Snapshot[] : []);
    } catch (error) {
      notify('danger', `Error loading snapshots: ${errorMessage(error)}`);
    }
  }

  async function createSnapshot() {
    try {
      await snapshotsCreate<true>({
        body: {
          ignoreUnavailable: createForm.ignoreUnavailable,
          includeGlobalState: createForm.includeGlobalState,
          indices: createForm.indices,
        },
        path: { ...clusterPath(connection), repository: createForm.repository, snapshot: createForm.snapshot },
        throwOnError: true,
      });
      notify('info', 'Snapshot successfully created');
      await loadSnapshots(repository);
    } catch (error) {
      notify('danger', `Error creating snapshot: ${errorMessage(error)}`);
    }
  }

  async function deleteSnapshot(snapshot: string) {
    try {
      await snapshotsDelete<true>({ path: { ...clusterPath(connection), repository, snapshot }, throwOnError: true });
      setDeleteSnapshotName('');
      notify('info', 'Snapshot successfully deleted');
      await loadSnapshots(repository);
    } catch (error) {
      notify('danger', `Error deleting snapshot: ${errorMessage(error)}`);
    }
  }

  async function restoreSnapshot(snapshot: string) {
    const form = restoreForms[snapshot] ?? defaultRestoreForm;
    try {
      await snapshotsRestore<true>({ body: form, path: { ...clusterPath(connection), repository, snapshot }, throwOnError: true });
      notify('info', 'Snapshot successfully restored');
    } catch (error) {
      notify('danger', `Error restoring snapshot: ${errorMessage(error)}`);
    }
  }

  const visibleIndices = showSpecial ? indices : indices.filter((index) => !index.special);
  const restoreSnapshotItem = snapshots.find((snapshot) => textValue(snapshot.snapshot) === restoreSnapshotName);
  const snapshotColumns: DataTableColumn<Snapshot>[] = [
    {
      header: <SortHeader name="name" sort={sort} onSort={(name) => setSort((value) => nextSort(value, name))}>snapshot</SortHeader>,
      key: 'name',
      render: (snapshot) => <><Icon name="database" /> {textValue(snapshot.snapshot)}</>,
    },
    {
      header: <SortHeader name="start_time" sort={sort} onSort={(name) => setSort((value) => nextSort(value, name))}>created</SortHeader>,
      key: 'start_time',
      render: (snapshot) => textValue(snapshot.start_time),
    },
    {
      header: <SortHeader name="state" sort={sort} onSort={(name) => setSort((value) => nextSort(value, name))}>state</SortHeader>,
      key: 'state',
      render: (snapshot) => textValue(snapshot.state),
    },
    {
      className: 'text-right',
      header: 'actions',
      headerClassName: 'text-right',
      key: 'actions',
      render: (snapshot) => {
        const snapshotName = textValue(snapshot.snapshot);
        return (
          <span className="inline-flex items-center justify-end gap-[10px]">
            <button
              aria-label={`restore snapshot ${snapshotName}`}
              className="btn btn-default btn-xs"
              title="restore snapshot"
              type="button"
              onClick={() => setRestoreSnapshotName((value) => (value === snapshotName ? '' : snapshotName))}
            >
              <Icon name="download" />
            </button>
            <button
              aria-label={`delete snapshot ${snapshotName}`}
              className="btn btn-danger btn-xs"
              title="delete snapshot"
              type="button"
              onClick={() => setDeleteSnapshotName(snapshotName)}
            >
              <Icon name="trash" />
            </button>
          </span>
        );
      },
    },
  ];

  return (
    <>
      {deleteSnapshotName ? (
        <ConfirmModal
          body={
            <>
              Delete snapshot <strong>{deleteSnapshotName}</strong> from repository <strong>{repository}</strong>? This operation cannot be undone.
            </>
          }
          confirmLabel={
            <>
              <Icon name="trash" /> delete snapshot
            </>
          }
          onClose={() => setDeleteSnapshotName('')}
          onConfirm={() => deleteSnapshot(deleteSnapshotName)}
          title="delete snapshot"
        />
      ) : null}
      <SplitPane
        storageKey="cerebro.snapshotsSplitPercent"
        left={
          <>
            <div className="flex items-center justify-between gap-[15px]">
              <h4>
                existing snapshots <small className="info-text">({snapshots.length})</small>
              </h4>
              <div className="form-inline form-group">
                <select className="form-control" value={repository} onChange={(event) => void loadSnapshots(event.target.value)}>
                  <option value="">select repository</option>
                  {repositories.map((repo) => <option key={repo}>{repo}</option>)}
                </select>
              </div>
            </div>
            <DataTable columns={snapshotColumns} getRowKey={(snapshot) => snapshot.uuid ?? textValue(snapshot.snapshot)} rows={sortByText(snapshots, sort, snapshotSortValue)} />
            {restoreSnapshotItem ? (
              <RestoreFormView
                form={restoreForms[restoreSnapshotName] ?? defaultRestoreForm}
                setForm={(form) => setRestoreForms((value) => ({ ...value, [restoreSnapshotName]: form }))}
                snapshot={restoreSnapshotItem}
                onRestore={() => void restoreSnapshot(restoreSnapshotName)}
              />
            ) : null}
          </>
        }
        right={
          <>
            <h4>create new snapshot</h4>
            <div className="row">
              <div className="col-sm-6">
                <FormGroup label="repository">
                  <select className="form-control" value={createForm.repository} onChange={(event) => setCreateForm((value) => ({ ...value, repository: event.target.value }))}>
                    <option value="">select repository</option>
                    {repositories.map((repo) => <option key={repo}>{repo}</option>)}
                  </select>
                </FormGroup>
                <FormGroup label="snapshot name">
                  <input className="form-control" placeholder="snapshot name" value={createForm.snapshot} onChange={(event) => setCreateForm((value) => ({ ...value, snapshot: event.target.value }))} />
                </FormGroup>
                <Checkbox checked={createForm.ignoreUnavailable} label="ignore unavailable indices" onChange={(checked) => setCreateForm((value) => ({ ...value, ignoreUnavailable: checked }))} />
                <Checkbox checked={createForm.includeGlobalState} label="include global state" onChange={(checked) => setCreateForm((value) => ({ ...value, includeGlobalState: checked }))} />
              </div>
              <div className="col-sm-6">
                <FormGroup label="indices (defaults to all if none is selected)">
                  <Checkbox checked={showSpecial} label="show special indices" onChange={setShowSpecial} />
                  <MultiSelect
                    options={visibleIndices.map((index) => index.name)}
                    value={createForm.indices}
                    onChange={(selected) => setCreateForm((value) => ({ ...value, indices: selected }))}
                  />
                </FormGroup>
              </div>
            </div>
            <div className="row">
              <div className="col-xs-12">
                <button className="btn btn-success pull-right" type="button" onClick={() => void createSnapshot()}>
                  <Icon name="file" /> create
                </button>
              </div>
            </div>
          </>
        }
      />
    </>
  );
}

type RestoreForm = {
  ignoreUnavailable: boolean;
  includeAliases: boolean;
  includeGlobalState: boolean;
  indices: string[];
  renamePattern?: string;
  renameReplacement?: string;
};

const defaultRestoreForm: RestoreForm = { ignoreUnavailable: true, includeAliases: true, includeGlobalState: true, indices: [] };

function RestoreFormView({ form, onRestore, setForm, snapshot }: { form: RestoreForm; onRestore: () => void; setForm: (form: RestoreForm) => void; snapshot: Snapshot }) {
  return (
    <div className="mt-[15px] border border-[#55595c] p-[12px]">
      <h4 className="!mt-0">restore {textValue(snapshot.snapshot)}</h4>
      <div className="row">
        <div className="col-lg-6">
          <FormGroup label="rename pattern">
            <input className="form-control" placeholder="index_(.+)" value={form.renamePattern ?? ''} onChange={(event) => setForm({ ...form, renamePattern: event.target.value })} />
          </FormGroup>
          <FormGroup label="rename replacement">
            <input className="form-control" placeholder="restored_index_$1" value={form.renameReplacement ?? ''} onChange={(event) => setForm({ ...form, renameReplacement: event.target.value })} />
          </FormGroup>
          <Checkbox checked={form.ignoreUnavailable} label="ignore unavailable indices" onChange={(checked) => setForm({ ...form, ignoreUnavailable: checked })} />
          <Checkbox checked={form.includeAliases} label="include aliases" onChange={(checked) => setForm({ ...form, includeAliases: checked })} />
          <Checkbox checked={form.includeGlobalState} label="include global state" onChange={(checked) => setForm({ ...form, includeGlobalState: checked })} />
        </div>
        <div className="col-lg-6">
          <FormGroup label="indices (defaults to all if none is selected)">
            <MultiSelect options={snapshot.indices ?? []} value={form.indices} onChange={(indices) => setForm({ ...form, indices })} />
          </FormGroup>
        </div>
      </div>
      <div className="row">
        <div className="col-lg-12 action-buttons">
          <button className="btn btn-success pull-right" type="button" onClick={onRestore}>
            <Icon name="download" /> restore
          </button>
        </div>
      </div>
    </div>
  );
}

function FormGroup({ children, label }: { children: ReactNode; label: string }) {
  return (
    <div className="form-group">
      <label className="form-label">{label}</label>
      {children}
    </div>
  );
}

function Checkbox({ checked, label, onChange }: { checked: boolean; label: string; onChange: (checked: boolean) => void }) {
  return (
    <div className="form-group">
      <label>
        <input checked={checked} type="checkbox" onChange={(event) => onChange(event.target.checked)} /> {label}
      </label>
    </div>
  );
}

function MultiSelect({ onChange, options, value }: { onChange: (value: string[]) => void; options: string[]; value: string[] }) {
  return (
    <select multiple className="form-control" size={13} value={value} onChange={(event) => onChange(Array.from(event.target.selectedOptions).map((option) => option.value))}>
      {options.sort().map((option) => <option key={option}>{option}</option>)}
    </select>
  );
}

function snapshotSortValue(snapshot: Snapshot, key: SnapshotSortKey) {
  switch (key) {
    case 'name':
      return textValue(snapshot.snapshot);
    case 'start_time':
      return textValue(snapshot.start_time);
    case 'state':
      return textValue(snapshot.state);
  }
}
