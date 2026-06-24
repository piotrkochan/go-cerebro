import { useEffect, useState } from 'react';
import { useForm } from '@tanstack/react-form';

import {
  repositoriesCreate,
  repositoriesDelete,
  repositoriesList,
  type HostBodyWritable,
  type Repository,
} from '../api/client';
import { Icon } from '../components/Icon';
import { LazyJsonEditor } from '../components/LazyJsonEditor';
import { ConfirmModal } from '../components/Modal';
import { SplitPane } from '../components/SplitPane';
import { repositoryFormDefaults, type RepositoryFormValues } from '../forms/repositoryForm';
import type { Notify } from '../types';
import { formatJson, parseJson, textValue } from '../utils/format';
import { nextSort, sortByText, type SortState } from '../utils/sort';

type RepositorySortKey = 'name' | 'type';

export function RepositoriesPage({
  connection,
  notify,
  refreshTick,
}: {
  connection: HostBodyWritable;
  notify: Notify;
  refreshTick: number;
}) {
  const [repositories, setRepositories] = useState<Repository[]>([]);
  const [update, setUpdate] = useState(false);
  const [deleteRepository, setDeleteRepository] = useState<Repository | null>(null);
  const [sort, setSort] = useState<SortState<RepositorySortKey>>({ key: 'name', order: 'asc' });
  const form = useForm({
    defaultValues: repositoryFormDefaults,
    onSubmit: async ({ value }) => {
      await save(value);
    },
  });

  useEffect(() => {
    async function load() {
      const result = await repositoriesList<true>({ body: connection, throwOnError: true });
      setRepositories(result.data.items ?? []);
    }
    void load();
  }, [connection, refreshTick]);

  async function save(values: RepositoryFormValues) {
    await repositoriesCreate<true>({
      body: { ...connection, name: values.name, settings: parseJson(values.settings) ?? {}, type: values.type },
      throwOnError: true,
    });
    notify('success', update ? 'repository updated' : 'repository created');
  }

  async function remove(repositoryName: string) {
    await repositoriesDelete<true>({ body: { ...connection, name: repositoryName }, throwOnError: true });
    setRepositories((value) => value.filter((repository) => repository.name !== repositoryName));
    setDeleteRepository(null);
    notify('success', 'repository removed');
  }

  return (
    <>
      {deleteRepository ? (
        <ConfirmModal
          body={
            <>
              Delete repository <strong>{deleteRepository.name}</strong>? This operation cannot be undone.
            </>
          }
          confirmLabel={
            <>
              <Icon name="trash" /> delete repository
            </>
          }
          onClose={() => setDeleteRepository(null)}
          onConfirm={() => remove(deleteRepository.name)}
          title="delete repository"
        />
      ) : null}
      <SplitPane
        storageKey="cerebro.repositoriesSplitPercent"
        left={
          <>
            <h4>existing repositories</h4>
            <table className="table">
              <thead>
                <tr>
                  <th>
                    <button className="normal-action border-0 bg-transparent p-0 text-inherit" type="button" onClick={() => setSort((value) => nextSort(value, 'name'))}>
                      name {sort.key === 'name' ? <Icon name={sort.order === 'asc' ? 'caret-down' : 'sort-alpha-desc'} /> : null}
                    </button>
                  </th>
                  <th>
                    <button className="normal-action border-0 bg-transparent p-0 text-inherit" type="button" onClick={() => setSort((value) => nextSort(value, 'type'))}>
                      type {sort.key === 'type' ? <Icon name={sort.order === 'asc' ? 'caret-down' : 'sort-alpha-desc'} /> : null}
                    </button>
                  </th>
                  <th className="text-right">actions</th>
                </tr>
              </thead>
              <tbody>
                {sortByText(repositories, sort, repositorySortValue).map((repository) => (
                  <tr key={repository.name}>
                    <td><Icon name="database" /> {repository.name}</td>
                    <td>{repository.type}</td>
                    <td className="text-right">
                      <span className="pull-right inline-flex items-center gap-[10px]">
                        <button
                          aria-label={`edit repository ${repository.name}`}
                          className="btn btn-default btn-xs"
                          title="edit repository"
                          type="button"
                          onClick={() => {
                            form.setFieldValue('name', repository.name);
                            form.setFieldValue('type', repository.type);
                            form.setFieldValue('settings', formatJson(repository.settings));
                            setUpdate(true);
                          }}
                        >
                          <Icon name="pencil" />
                        </button>
                        <button
                          aria-label={`delete repository ${repository.name}`}
                          className="btn btn-danger btn-xs"
                          title="delete repository"
                          type="button"
                          onClick={() => setDeleteRepository(repository)}
                        >
                          <Icon name="trash" />
                        </button>
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </>
        }
        right={
          <>
            <h4>{update ? 'update repository' : 'create new repository'}</h4>
            <div className="row">
              <div className="form-group col-md-6">
                <label className="form-label">repository name</label>
                <form.Field name="name">
                  {(field) => <input className="form-control" placeholder="repository name" value={field.state.value} onChange={(event) => field.handleChange(event.target.value)} />}
                </form.Field>
              </div>
              <div className="form-group col-md-6">
                <label className="form-label">type</label>
                <form.Field name="type">
                  {(field) => (
                    <select className="form-control" value={field.state.value} onChange={(event) => field.handleChange(event.target.value)}>
                      {['fs', 'url', 's3', 'gcs', 'hdfs', 'azure'].map((value) => <option key={value}>{value}</option>)}
                    </select>
                  )}
                </form.Field>
              </div>
            </div>
            <div className="row">
              <div className="col-xs-12 form-group">
                <label className="form-label">settings</label>
                <form.Field name="settings">
                  {(field) => <LazyJsonEditor height={260} value={field.state.value} onChange={field.handleChange} />}
                </form.Field>
              </div>
            </div>
            <div className="row">
              <div className="col-lg-12 action-buttons">
                <button className="btn btn-success pull-right" type="button" onClick={() => void form.handleSubmit()}>
                  <Icon name={update ? 'save' : 'file'} /> {update ? 'update' : 'create'}
                </button>
              </div>
            </div>
          </>
        }
      />
    </>
  );
}

function repositorySortValue(repository: Repository, key: RepositorySortKey) {
  return key === 'name' ? repository.name : textValue(repository.type);
}
