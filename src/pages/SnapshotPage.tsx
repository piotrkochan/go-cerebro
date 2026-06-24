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
import { Icon } from '../components/Icon';
import type { Notify } from '../types';
import { errorMessage, textValue } from '../utils/format';

type SnapshotIndex = { name: string; special?: boolean };
type Snapshot = { indices?: string[]; snapshot?: string; start_time?: unknown; state?: unknown; uuid?: string };
type SnapshotLoad = { indices?: SnapshotIndex[]; repositories?: string[] };

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
  const [createForm, setCreateForm] = useState({ ignoreUnavailable: false, includeGlobalState: true, indices: [] as string[], repository: '', snapshot: '' });
  const [restoreForms, setRestoreForms] = useState<Record<string, RestoreForm>>({});

  useEffect(() => {
    async function load() {
      try {
        const result = await snapshotsGet<true>({ body: connection, throwOnError: true });
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
      const result = await snapshotsLoad<true>({ body: { ...connection, repository: nextRepository }, throwOnError: true });
      setSnapshots(Array.isArray(result.data.data) ? result.data.data as Snapshot[] : []);
    } catch (error) {
      notify('danger', `Error loading snapshots: ${errorMessage(error)}`);
    }
  }

  async function createSnapshot() {
    try {
      await snapshotsCreate<true>({ body: { ...connection, ...createForm }, throwOnError: true });
      notify('info', 'Snapshot successfully created');
      await loadSnapshots(repository);
    } catch (error) {
      notify('danger', `Error creating snapshot: ${errorMessage(error)}`);
    }
  }

  async function deleteSnapshot(snapshot: string) {
    try {
      await snapshotsDelete<true>({ body: { ...connection, repository, snapshot }, throwOnError: true });
      notify('info', 'Snapshot successfully deleted');
      await loadSnapshots(repository);
    } catch (error) {
      notify('danger', `Error deleting snapshot: ${errorMessage(error)}`);
    }
  }

  async function restoreSnapshot(snapshot: string) {
    const form = restoreForms[snapshot] ?? defaultRestoreForm;
    try {
      await snapshotsRestore<true>({ body: { ...connection, repository, snapshot, ...form }, throwOnError: true });
      notify('info', 'Snapshot successfully restored');
    } catch (error) {
      notify('danger', `Error restoring snapshot: ${errorMessage(error)}`);
    }
  }

  const visibleIndices = showSpecial ? indices : indices.filter((index) => !index.special);

  return (
    <div className="row">
      <div className="col-md-6">
        <h4>
          existing snapshots
          <div className="form-inline form-group pull-right">
            <select className="form-control" value={repository} onChange={(event) => void loadSnapshots(event.target.value)}>
              <option value="">select repository</option>
              {repositories.map((repo) => <option key={repo}>{repo}</option>)}
            </select>
          </div>
        </h4>
        <table className="table">
          <tbody>
            {snapshots.map((snapshot) => {
              const snapshotName = textValue(snapshot.snapshot);
              return (
                <tr key={snapshot.uuid ?? snapshotName}>
                  <td>
                    <div>
                      <Icon name="database" /> {snapshotName}
                      <span className="info-text"> created at </span> {textValue(snapshot.start_time)}
                      <span className="info-text"> state </span> {textValue(snapshot.state)}
                      <Icon className="normal-action alert-danger pull-right" name="trash" onClick={() => void deleteSnapshot(snapshotName)} />
                      <Icon className="normal-action pull-right" name="download" />
                    </div>
                    <RestoreFormView
                      form={restoreForms[snapshotName] ?? defaultRestoreForm}
                      setForm={(form) => setRestoreForms((value) => ({ ...value, [snapshotName]: form }))}
                      snapshot={snapshot}
                      onRestore={() => void restoreSnapshot(snapshotName)}
                    />
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      <div className="col-md-6">
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
              <label className="cluster-map-node-type">
                <input checked={showSpecial} type="checkbox" onChange={(event) => setShowSpecial(event.target.checked)} /> show special indices
              </label>
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
            <button className="btn btn-success pull-right" type="submit" onClick={() => void createSnapshot()}>
              <Icon name="file" /> create
            </button>
          </div>
        </div>
      </div>
    </div>
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
    <div>
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
          <button className="btn btn-success pull-right" type="submit" onClick={onRestore}>
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
