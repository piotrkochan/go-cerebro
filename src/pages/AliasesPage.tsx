import { useEffect, useState } from 'react';
import { useForm } from '@tanstack/react-form';

import { aliasesGet, aliasesUpdate, overview, type Alias, type HostBodyWritable, type Overview } from '../api/client';
import { Icon } from '../components/Icon';
import { LazyJsonEditor } from '../components/LazyJsonEditor';
import { aliasFormDefaults, type AliasFormValues } from '../forms/aliasForm';
import type { Notify } from '../types';
import { formatJson, parseJson, textValue } from '../utils/format';

export function AliasesPage({
  connection,
  notify,
  refreshTick,
}: {
  connection: HostBodyWritable;
  notify: Notify;
  refreshTick: number;
}) {
  const [aliases, setAliases] = useState<Alias[]>([]);
  const [overviewData, setOverviewData] = useState<Overview>();
  const [filter, setFilter] = useState({ alias: '', index: '' });
  const [changes, setChanges] = useState<unknown[]>([]);
  const form = useForm({
    defaultValues: aliasFormDefaults,
    onSubmit: ({ value }) => {
      addAlias(value);
    },
  });

  useEffect(() => {
    async function load() {
      const [aliasesResult, overviewResult] = await Promise.all([
        aliasesGet<true>({ body: connection, throwOnError: true }),
        overview<true>({ body: connection, throwOnError: true }),
      ]);
      setAliases(aliasesResult.data.items ?? []);
      setOverviewData(overviewResult.data);
    }
    void load();
  }, [connection, refreshTick]);

  const filtered = aliases.filter(
    (alias) =>
      alias.alias.toLowerCase().includes(filter.alias.toLowerCase()) &&
      alias.index.toLowerCase().includes(filter.index.toLowerCase()),
  );

  function removeIndexAlias(alias: Alias) {
    setChanges((value) => [...value, { remove: { alias: alias.alias, index: alias.index } }]);
  }

  function addAlias(values: AliasFormValues) {
    setChanges((value) => [
      ...value,
      {
        add: {
          alias: values.alias,
          filter: parseJson(values.filter),
          index: values.index,
          index_routing: values.index_routing || undefined,
          search_routing: values.search_routing || undefined,
        },
      },
    ]);
    form.setFieldValue('alias', '');
    form.setFieldValue('filter', '');
    form.setFieldValue('index', '');
    form.setFieldValue('index_routing', '');
    form.setFieldValue('search_routing', '');
  }

  async function saveChanges() {
    await aliasesUpdate<true>({ body: { ...connection, changes }, throwOnError: true });
    setChanges([]);
    notify('success', 'aliases updated');
  }

  return (
    <div className="row">
      <div className="col-md-6">
        <h4>current aliases</h4>
        <div className="row">
          <div className="col-md-6">
            {aliases.length ? (
              <div className="row">
                <div className="col-md-6 form-group">
                  <input className="form-control" placeholder="filter by alias" value={filter.alias} onChange={(event) => setFilter((value) => ({ ...value, alias: event.target.value }))} />
                </div>
                <div className="col-md-6 form-group">
                  <input className="form-control" placeholder="filter by index" value={filter.index} onChange={(event) => setFilter((value) => ({ ...value, index: event.target.value }))} />
                </div>
              </div>
            ) : null}
          </div>
          <div className="col-xs-12">
            <table className="table">
              <tbody>
                {filtered.map((alias, index) => (
                  <tr key={`${alias.alias}-${alias.index}-${index}`}>
                    <td>
                      <span>
                        <Icon name="tag" /> {alias.alias} <span className="info-text"> assigned to index</span>{' '}
                        {alias.index}
                        {alias.search_routing ? <><span className="info-text"> with search routing</span> {textValue(alias.search_routing)}</> : null}
                        {alias.index_routing ? <><span className="info-text"> with index routing</span> {textValue(alias.index_routing)}</> : null}
                        <Icon className="normal-action alert-danger pull-right" name="trash" onClick={() => removeIndexAlias(alias)} />
                      </span>
                      {alias.filter ? <pre>{formatJson(alias.filter)}</pre> : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
      <div className="col-md-6">
        <h4>changes</h4>
        <div className="row">
          <div className="col-xs-12">
            <div className="form-inline form-group">
              <div className="form-group">
                <form.Field name="alias">
                  {(field) => <input className="form-control" placeholder="alias" value={field.state.value} onChange={(event) => field.handleChange(event.target.value)} />}
                </form.Field>
              </div>
              <span className="info-text"> assigned to index </span>
              <div className="form-group">
                <form.Field name="index">
                  {(field) => (
                    <select className="form-control" value={field.state.value} onChange={(event) => field.handleChange(event.target.value)}>
                      <option value="">select index</option>
                      {(overviewData?.indices ?? []).map((index) => <option key={index.name}>{index.name}</option>)}
                    </select>
                  )}
                </form.Field>
              </div>
              <div className="form-group pull-right">
                <button className="btn btn-info" type="button" onClick={() => void form.handleSubmit()}>
                  <Icon className="normal-action alert-info" name="plus" />
                </button>
              </div>
            </div>
            <div className="col-xs-12 form-group">
              <form.Field name="filter">
                {(field) => <LazyJsonEditor height={225} value={field.state.value} onChange={field.handleChange} />}
              </form.Field>
            </div>
          </div>
        </div>
        <table className="table">
          <tbody>
            {changes.map((change, index) => (
              <tr key={index}>
                <td>
                  <pre>{formatJson(change)}</pre>
                  <span className="pull-right"><Icon className="normal-action" name="undo" onClick={() => setChanges((value) => value.filter((_, i) => i !== index))} /></span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        <div className="text-right">
          {changes.length ? <button className="btn btn-success" type="button" onClick={() => void saveChanges()}>apply</button> : null}
        </div>
      </div>
    </div>
  );
}
