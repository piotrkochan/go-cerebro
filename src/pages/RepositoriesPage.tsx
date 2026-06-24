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
import { repositoryFormDefaults, type RepositoryFormValues } from '../forms/repositoryForm';
import type { Notify } from '../types';
import { formatJson, parseJson } from '../utils/format';

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
    notify('success', 'repository removed');
  }

  return (
    <div className="row">
      <div className="col-md-6">
        <h4>existing repositories</h4>
        <table className="table">
          <tbody>
            {repositories.map((repository) => (
              <tr key={repository.name}>
                <td>
                  <Icon name="database" /> {repository.name} <span className="info-text"> of type </span>{' '}
                  {repository.type}
                  <Icon className="normal-action alert-danger pull-right" name="trash" onClick={() => void remove(repository.name)} />
                  <Icon className="normal-action pull-right" name="pencil" onClick={() => {
                    form.setFieldValue('name', repository.name);
                    form.setFieldValue('type', repository.type);
                    form.setFieldValue('settings', formatJson(repository.settings));
                    setUpdate(true);
                  }} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="col-md-6">
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
      </div>
    </div>
  );
}
