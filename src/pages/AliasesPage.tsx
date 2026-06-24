import { useEffect, useState } from 'react';
import { useForm } from '@tanstack/react-form';

import { aliasesGet, aliasesUpdate, overview, type Alias, type HostBodyWritable, type Overview } from '../api/client';
import { Icon } from '../components/Icon';
import { LazyJsonEditor } from '../components/LazyJsonEditor';
import { ConfirmModal } from '../components/Modal';
import { SplitPane } from '../components/SplitPane';
import { aliasFormDefaults, type AliasFormValues } from '../forms/aliasForm';
import type { Notify } from '../types';
import { formatJson, textValue } from '../utils/format';
import { nextSort, sortByText, type SortState } from '../utils/sort';

type AliasSortKey = 'alias' | 'index' | 'routing';

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
  const [deleteConfirm, setDeleteConfirm] = useState<Alias | null>(null);
  const [sort, setSort] = useState<SortState<AliasSortKey>>({ key: 'alias', order: 'asc' });
  const form = useForm({
    defaultValues: aliasFormDefaults,
    onSubmit: ({ value }) => {
      queueAliasChange(value);
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
      resetAliasForm();
    }
    void load();
  }, [connection, refreshTick]);

  const filtered = sortByText(
    aliases.filter(
      (alias) =>
        alias.alias.toLowerCase().includes(filter.alias.toLowerCase()) &&
        alias.index.toLowerCase().includes(filter.index.toLowerCase()),
    ),
    sort,
    aliasSortValue,
  );

  function removeIndexAlias(alias: Alias) {
    setChanges((value) => [...value, { remove: { alias: alias.alias, index: alias.index } }]);
    setDeleteConfirm(null);
  }

  function queueAliasChange(values: AliasFormValues) {
    const alias = values.alias.trim();
    const index = values.index.trim();
    if (!alias || !index) {
      notify('danger', 'alias and index are required');
      return;
    }
    let addAction: unknown;
    try {
      addAction = aliasAddAction({ ...values, alias, index });
    } catch (error) {
      notify('danger', error instanceof Error ? error.message : 'invalid alias filter JSON');
      return;
    }
    setChanges((value) => [...value, addAction]);
    resetAliasForm();
  }

  function resetAliasForm() {
    form.setFieldValue('alias', '');
    form.setFieldValue('filter', '');
    form.setFieldValue('index', '');
    form.setFieldValue('index_routing', '');
    form.setFieldValue('search_routing', '');
  }

  async function saveChanges() {
    await aliasesUpdate<true>({ body: { ...connection, changes }, throwOnError: true });
    setChanges([]);
    resetAliasForm();
    notify('success', 'aliases updated');
  }

  return (
    <>
      {deleteConfirm ? (
        <ConfirmModal
          body={
            <>
              Remove alias <strong>{deleteConfirm.alias}</strong> from index <strong>{deleteConfirm.index}</strong>?
              <div className="info-text">Elasticsearch is changed only after apply changes.</div>
            </>
          }
          confirmLabel={
            <>
              <Icon name="trash" /> remove alias
            </>
          }
          onClose={() => setDeleteConfirm(null)}
          onConfirm={() => removeIndexAlias(deleteConfirm)}
          title="remove alias"
        />
      ) : null}
      <SplitPane
        storageKey="cerebro.aliasesSplitPercent"
        left={
          <>
          <h4>
            current aliases <small className="info-text">({filtered.length})</small>
          </h4>
          <div className="row">
            <div className="col-sm-6 form-group">
              <input className="form-control" placeholder="filter by alias" value={filter.alias} onChange={(event) => setFilter((value) => ({ ...value, alias: event.target.value }))} />
            </div>
            <div className="col-sm-6 form-group">
              <input className="form-control" placeholder="filter by index" value={filter.index} onChange={(event) => setFilter((value) => ({ ...value, index: event.target.value }))} />
            </div>
            <div className="col-xs-12">
              {filtered.length ? (
                <table className="table table-condensed">
                  <thead>
                    <tr>
                      <th>
                        <button className="normal-action border-0 bg-transparent p-0 text-inherit" type="button" onClick={() => setSort((value) => nextSort(value, 'alias'))}>
                          alias {sort.key === 'alias' ? <Icon name={sort.order === 'asc' ? 'caret-down' : 'sort-alpha-desc'} /> : null}
                        </button>
                      </th>
                      <th>
                        <button className="normal-action border-0 bg-transparent p-0 text-inherit" type="button" onClick={() => setSort((value) => nextSort(value, 'index'))}>
                          index {sort.key === 'index' ? <Icon name={sort.order === 'asc' ? 'caret-down' : 'sort-alpha-desc'} /> : null}
                        </button>
                      </th>
                      <th>
                        <button className="normal-action border-0 bg-transparent p-0 text-inherit" type="button" onClick={() => setSort((value) => nextSort(value, 'routing'))}>
                          routing / filter {sort.key === 'routing' ? <Icon name={sort.order === 'asc' ? 'caret-down' : 'sort-alpha-desc'} /> : null}
                        </button>
                      </th>
                      <th className="text-right">remove</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filtered.map((alias, index) => (
                      <tr key={`${alias.alias}-${alias.index}-${index}`}>
                        <td>
                          <Icon name="tag" /> {alias.alias}
                        </td>
                        <td>{alias.index}</td>
                        <td>
                          {alias.search_routing ? <div><span className="info-text">search:</span> {textValue(alias.search_routing)}</div> : null}
                          {alias.index_routing ? <div><span className="info-text">index:</span> {textValue(alias.index_routing)}</div> : null}
                          {alias.filter ? (
                            <details>
                              <summary className="normal-action info-text">filter JSON</summary>
                              <pre>{formatJson(alias.filter)}</pre>
                            </details>
                          ) : (
                            <span className="info-text">none</span>
                          )}
                        </td>
                        <td className="text-right">
                          <div className="inline-flex items-center justify-end gap-[10px]">
                            <button aria-label={`remove alias ${alias.alias}`} className="btn btn-danger btn-xs" title="remove alias" type="button" onClick={() => setDeleteConfirm(alias)}>
                              <Icon name="trash" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : (
                <div className="info-text">no aliases match current filters</div>
              )}
            </div>
          </div>
          </>
        }
        right={
          <>
          <h4>add alias</h4>
          <div className="subtitle">
            add alias details to changes
          </div>
          <div className="row">
            <div className="col-xs-12">
              <div className="row">
                <div className="col-sm-6 form-group">
                  <label>alias</label>
                  <form.Field name="alias">
                    {(field) => <input className="form-control" placeholder="alias" value={field.state.value} onChange={(event) => field.handleChange(event.target.value)} />}
                  </form.Field>
                </div>
                <div className="col-sm-6 form-group">
                  <label>assigned index</label>
                  <form.Field name="index">
                    {(field) => (
                      <select className="form-control" value={field.state.value} onChange={(event) => field.handleChange(event.target.value)}>
                        <option value="">select index</option>
                        {(overviewData?.indices ?? []).map((index) => <option key={index.name}>{index.name}</option>)}
                      </select>
                    )}
                  </form.Field>
                </div>
              </div>
              <div className="row">
                <div className="col-sm-6 form-group">
                  <label>search routing</label>
                  <form.Field name="search_routing">
                    {(field) => (
                      <input
                        className="form-control"
                        placeholder="optional"
                        value={field.state.value}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                    )}
                  </form.Field>
                </div>
                <div className="col-sm-6 form-group">
                  <label>index routing</label>
                  <form.Field name="index_routing">
                    {(field) => (
                      <input
                        className="form-control"
                        placeholder="optional"
                        value={field.state.value}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                    )}
                  </form.Field>
                </div>
              </div>
              <div className="form-group">
                <label>filter JSON</label>
                <div className="subtitle">optional Elasticsearch alias filter</div>
                <form.Field name="filter">
                  {(field) => <LazyJsonEditor height={225} value={field.state.value} onChange={field.handleChange} />}
                </form.Field>
              </div>
              <div className="form-group text-right">
                <button className="btn btn-info" type="button" onClick={() => void form.handleSubmit()}>
                  <Icon name="plus" /> add
                </button>
              </div>
            </div>
          </div>
          </>
        }
      />
      <div className="row">
        <div className="col-xs-12">
          <h4>
            changes <small className="info-text">({changes.length})</small>
          </h4>
          {changes.length ? (
            <>
              <table className="table table-condensed">
                <tbody>
                  {changes.map((change, index) => (
                    <tr key={index}>
                      <td className="col-xs-2">{aliasChangeLabel(change)}</td>
                      <td>
                        <pre>{formatJson(change)}</pre>
                      </td>
                      <td className="text-right col-xs-1">
                        <button className="btn btn-default btn-xs" type="button" onClick={() => setChanges((value) => value.filter((_, i) => i !== index))}>
                          <Icon name="undo" /> undo
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <div className="text-right">
                <button className="btn btn-default" type="button" onClick={() => setChanges([])}>
                  discard all
                </button>{' '}
                <button className="btn btn-success" type="button" onClick={() => void saveChanges()}>
                  <Icon name="check" /> apply
                </button>
              </div>
            </>
          ) : (
            <div className="info-text">no pending changes</div>
          )}
        </div>
      </div>
    </>
  );
}

function aliasAddAction(values: AliasFormValues) {
  const filter = parseAliasFilter(values.filter);
  const add: Record<string, unknown> = {
    alias: values.alias,
    index: values.index,
  };
  if (filter !== undefined) add.filter = filter;
  if (values.index_routing.trim()) add.index_routing = values.index_routing.trim();
  if (values.search_routing.trim()) add.search_routing = values.search_routing.trim();
  return { add };
}

function aliasSortValue(alias: Alias, key: AliasSortKey) {
  switch (key) {
    case 'alias':
      return alias.alias;
    case 'index':
      return alias.index;
    case 'routing':
      return `${textValue(alias.search_routing)} ${textValue(alias.index_routing)} ${alias.filter ? formatJson(alias.filter) : ''}`;
  }
}

function parseAliasFilter(value: string) {
  if (!value.trim()) return undefined;
  try {
    return JSON.parse(value);
  } catch {
    throw new Error('filter JSON is invalid');
  }
}

function aliasChangeLabel(change: unknown) {
  if (typeof change !== 'object' || change === null) return 'change';
  const value = change as { add?: unknown; remove?: unknown };
  if (value.add) return 'add';
  if (value.remove) return 'remove';
  return 'change';
}
