import { useEffect, useRef, useState } from 'react';
import { useStore } from '@tanstack/react-store';

import {
  commonsGetIndexMapping,
  commonsGetIndexSettings,
  dataExplorerSave,
  dataExplorerSearch,
  type HostBodyWritable,
} from '../api/client';
import { Icon } from '../components/Icon';
import { LazyJsonEditor } from '../components/LazyJsonEditor';
import { Loading } from '../components/LegacyUi';
import { dataExplorerActions, dataExplorerStore } from '../stores/dataExplorerStore';
import type { Notify } from '../types';
import { errorMessage, formatJson, formatNumber, textValue } from '../utils/format';

type DataRow = Record<string, unknown>;
type EditorMode = 'edit' | 'insert' | 'view';

type EditorState = {
  documentID: string;
  mode: EditorMode;
  value: string;
};

type QuerySuggestion = {
  detail: string;
  label: string;
  value: string;
};

export function DataExplorerPage({
  connection,
  enabled,
  index,
  notify,
}: {
  connection: HostBodyWritable;
  enabled: boolean;
  index: string;
  notify: Notify;
}) {
  const prefs = useStore(dataExplorerStore);
  const [columns, setColumns] = useState<string[]>([]);
  const [editor, setEditor] = useState<EditorState | null>(null);
  const [executedQuery, setExecutedQuery] = useState('');
  const [fields, setFields] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(0);
  const [query, setQuery] = useState('');
  const [queryCursor, setQueryCursor] = useState(0);
  const [queryFocused, setQueryFocused] = useState(false);
  const [readOnly, setReadOnly] = useState(false);
  const [rows, setRows] = useState<DataRow[]>([]);
  const [saving, setSaving] = useState(false);
  const [total, setTotal] = useState(0);
  const queryInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!enabled || !index) return;
    void load();
  }, [connection.host, enabled, executedQuery, index, page, prefs.pageSize, prefs.queryMode, prefs.sortField, prefs.sortOrder]);

  useEffect(() => {
    if (!enabled || !index) return;
    void loadIndexMetadata();
  }, [connection.host, enabled, index]);

  useEffect(() => {
    if (!editor) return;
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') setEditor(null);
    }
    window.addEventListener('keydown', closeOnEscape);
    return () => window.removeEventListener('keydown', closeOnEscape);
  }, [editor]);

  async function load() {
    setLoading(true);
    try {
      const result = await dataExplorerSearch<true>({
        body: {
          ...connection,
          index,
          page,
          query: executedQuery,
          query_mode: prefs.queryMode,
          size: prefs.pageSize,
          sort_field: prefs.sortField,
          sort_order: prefs.sortOrder,
        },
        throwOnError: true,
      });
      setColumns(result.data.columns ?? []);
      setRows((result.data.rows ?? []) as DataRow[]);
      setTotal(result.data.total ?? 0);
    } catch (error) {
      setColumns([]);
      setRows([]);
      setTotal(0);
      notify('danger', `Error loading data: ${errorMessage(error)}`);
    } finally {
      setLoading(false);
    }
  }

  async function loadIndexMetadata() {
    try {
      const [mapping, settings] = await Promise.all([
        commonsGetIndexMapping<true>({ body: { ...connection, index }, throwOnError: true }),
        commonsGetIndexSettings<true>({ body: { ...connection, index }, throwOnError: true }),
      ]);
      setFields(extractMappingFields(mapping.data.data, index));
      setReadOnly(indexSettingsReadOnly(settings.data.data, index));
    } catch {
      setFields([]);
      setReadOnly(false);
    }
  }

  function submitQuery() {
    setPage(0);
    setExecutedQuery(query);
  }

  function clearQuery() {
    setQuery('');
    setPage(0);
    setExecutedQuery('');
  }

  function insertSuggestion(suggestion: QuerySuggestion) {
    const cursor = queryInputRef.current?.selectionStart ?? queryCursor;
    const { end, start } = queryTokenRange(query, cursor);
    const next = `${query.slice(0, start)}${suggestion.value}${query.slice(end)}`;
    const nextCursor = start + suggestion.value.length;
    setQuery(next);
    setQueryCursor(nextCursor);
    setQueryFocused(false);
    window.requestAnimationFrame(() => {
      queryInputRef.current?.focus();
      queryInputRef.current?.setSelectionRange(nextCursor, nextCursor);
    });
  }

  function openEditor(mode: EditorMode, row?: DataRow) {
    if (readOnly && (mode === 'edit' || mode === 'insert')) {
      notify('info', 'Index is read-only');
      return;
    }
    setEditor({
      documentID: mode === 'insert' ? '' : textValue(row?._id),
      mode,
      value: formatJson(mode === 'insert' ? {} : sourceValue(row)),
    });
  }

  function openRow(row: DataRow) {
    if (window.getSelection()?.toString()) return;
    openEditor('view', row);
  }

  async function saveDocument() {
    if (!editor) return;
    if (readOnly) {
      notify('info', 'Index is read-only');
      return;
    }
    setSaving(true);
    try {
      await dataExplorerSave<true>({
        body: {
          ...connection,
          document: JSON.parse(editor.value) as unknown,
          id: editor.documentID,
          index,
        },
        throwOnError: true,
      });
      notify('success', editor.mode === 'insert' ? 'Document inserted' : 'Document saved');
      setEditor(null);
      await load();
    } catch (error) {
      notify('danger', `Error saving document: ${errorMessage(error)}`);
    } finally {
      setSaving(false);
    }
  }

  if (!enabled) {
    return <h4 className="text-center">Data explorer is disabled</h4>;
  }

  if (!index) {
    return <h4 className="text-center">No index selected</h4>;
  }

  const first = total > 0 ? page * prefs.pageSize + 1 : 0;
  const last = Math.min(total, (page + 1) * prefs.pageSize);
  const hasPrevious = page > 0;
  const hasNext = (page + 1) * prefs.pageSize < total;
  const queryChanged = query !== executedQuery;
  const suggestions = querySuggestions(query, queryCursor, prefs.queryMode, fields.length ? fields : columns);

  return (
    <>
      <div className="row">
        <div className="col-xs-12">
          <h4 className="mb-[10px]">
            <Icon name="database" /> data explorer: {index}
          </h4>
        </div>
      </div>

      <div className="row">
        <div className="col-xs-12">
          <div className="mb-[10px] flex flex-wrap items-center gap-[8px] border border-[#55595c] bg-[#373a3c] px-[10px] py-[8px]">
            <span className="info-text whitespace-nowrap">query</span>
            <select
              className="h-[30px] border border-[#55595c] bg-[#2b2d2f] px-[6px] text-[#eceeef] outline-none focus:border-[#1ca8dd]"
              title="query language"
              value={prefs.queryMode}
              onChange={(event) => dataExplorerActions.setQueryMode(event.target.value === 'lucene' ? 'lucene' : 'kql')}
            >
              <option value="kql">KQL</option>
              <option value="lucene">Lucene</option>
            </select>
            <div className="relative min-w-[260px] flex-1">
              <input
                ref={queryInputRef}
                aria-label="Document query"
                className="h-[30px] w-full border border-[#55595c] bg-[#2b2d2f] px-[8px] text-[#eceeef] outline-none focus:border-[#1ca8dd]"
                placeholder={prefs.queryMode === 'kql' ? 'items_count: 7 and status: paid' : 'items_count:7 AND status:paid'}
                value={query}
                onBlur={() => window.setTimeout(() => setQueryFocused(false), 120)}
                onChange={(event) => {
                  setQuery(event.target.value);
                  setQueryCursor(event.target.selectionStart ?? event.target.value.length);
                }}
                onClick={(event) => setQueryCursor(event.currentTarget.selectionStart ?? query.length)}
                onFocus={(event) => {
                  setQueryFocused(true);
                  setQueryCursor(event.currentTarget.selectionStart ?? query.length);
                }}
                onKeyUp={(event) => setQueryCursor(event.currentTarget.selectionStart ?? query.length)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') submitQuery();
                  if (event.key === 'Escape') clearQuery();
                }}
              />
              {queryFocused && suggestions.length ? (
                <div className="absolute left-0 right-0 top-full z-[1100] mt-1 max-h-64 overflow-auto border border-[#55595c] bg-[#373a3c] shadow-lg">
                  {suggestions.map((suggestion) => (
                    <button
                      className="flex w-full cursor-pointer items-center justify-between gap-3 px-3 py-1.5 text-left text-[#eceeef] hover:bg-[#434749] hover:text-white"
                      key={`${suggestion.detail}-${suggestion.label}`}
                      type="button"
                      onMouseDown={(event) => {
                        event.preventDefault();
                        insertSuggestion(suggestion);
                      }}
                    >
                      <span className="truncate">{suggestion.label}</span>
                      <span className="info-text whitespace-nowrap">{suggestion.detail}</span>
                    </button>
                  ))}
                </div>
              ) : null}
            </div>
            {queryChanged ? (
              <button className="btn btn-default btn-xs" disabled={loading} type="button" onClick={submitQuery}>
                apply
              </button>
            ) : null}
            {query ? (
              <button className="btn btn-default btn-xs" disabled={loading} type="button" onClick={clearQuery}>
                clear
              </button>
            ) : null}
            <span className="info-text whitespace-nowrap">rows</span>
            <select
              className="h-[30px] border border-[#55595c] bg-[#2b2d2f] px-[6px] text-[#eceeef] outline-none focus:border-[#1ca8dd]"
              value={prefs.pageSize}
              onChange={(event) => {
                setPage(0);
                dataExplorerActions.setPageSize(Number(event.target.value));
              }}
            >
              {[10, 25, 50, 100].map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
            </select>
            <button className="btn btn-default btn-xs" disabled={loading} title="refresh" type="button" onClick={() => void load()}>
              <Icon name={loading ? 'spinner' : 'refresh'} spin={loading} />
            </button>
            {readOnly ? <span className="label label-warning">read-only</span> : null}
            <button
              className="btn btn-default btn-xs"
              disabled={readOnly}
              title={readOnly ? 'index is read-only' : 'insert document'}
              type="button"
              onClick={() => openEditor('insert')}
            >
              <Icon name="plus" /> insert
            </button>
            <span className="info-text ml-auto whitespace-nowrap">
              {prefs.sortField ? ` | ${prefs.sortField} ${prefs.sortOrder}` : ''}
            </span>
            <DataPager
              first={first}
              hasNext={hasNext}
              hasPrevious={hasPrevious}
              last={last}
              setPage={setPage}
              page={page}
              total={total}
            />
          </div>
        </div>
      </div>

      {loading ? <Loading label="loading documents..." /> : null}
      {!loading && rows.length === 0 ? <h4 className="text-center">No data available</h4> : null}
      {rows.length ? (
        <div className="row">
          <div className="col-xs-12">
            <div className="border border-[#55595c]">
              <table className="table table-condensed table-hover mb-0">
                <thead>
                  <tr>
                    {columns.map((column) => (
                      <th
                        className={`sticky top-0 z-[1] whitespace-nowrap bg-[#373a3c] ${column === '_id' ? '' : 'normal-action'}`}
                        key={column}
                        onClick={() => column !== '_id' && dataExplorerActions.setSort(column)}
                      >
                        {column}
                        {prefs.sortField === column ? <Icon name={prefs.sortOrder === 'asc' ? 'caret-down' : 'sort-alpha-desc'} /> : null}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row, rowIndex) => (
                    <tr
                      className="normal-action"
                      key={`${textValue(row._id)}-${rowIndex}`}
                      onClick={() => openRow(row)}
                    >
                      {columns.map((column) => (
                        <td className="max-w-[280px] truncate whitespace-nowrap" key={column} title={textValue(row[column])}>
                          {textValue(row[column])}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      ) : null}
      {rows.length ? (
        <div className="row">
          <div className="col-xs-12">
            <div className="mt-[10px] flex justify-end">
              <DataPager
                first={first}
                hasNext={hasNext}
                hasPrevious={hasPrevious}
                last={last}
                setPage={setPage}
                page={page}
                total={total}
              />
            </div>
          </div>
        </div>
      ) : null}

      {editor ? (
        <>
          <div className="modal-backdrop in" onClick={() => setEditor(null)} />
          <div className="modal in !block" tabIndex={-1}>
            <div className="modal-dialog modal-lg">
              <div className="modal-content">
                <div className="modal-header flex items-center justify-between">
                  <h4 className="modal-title !m-0">
                    <Icon name={editor.mode === 'insert' ? 'plus' : editor.mode === 'edit' ? 'edit' : 'code'} />{' '}
                    {editor.mode === 'insert' ? 'insert document' : editor.mode === 'edit' ? 'edit document' : 'document'}
                    {editor.documentID ? <span className="info-text"> {editor.documentID}</span> : null}
                  </h4>
                  <button
                    aria-label="close"
                    className="close !float-none !text-[#eceeef] !opacity-[.85] [text-shadow:none] hover:!text-white hover:!opacity-100 focus:!text-white focus:!opacity-100"
                    type="button"
                    onClick={() => setEditor(null)}
                  >
                    &times;
                  </button>
                </div>
                <div className="modal-body">
                  {editor.mode === 'insert' ? (
                    <div className="form-group">
                      <label>document id</label>
                      <input
                        className="form-control"
                        placeholder="empty = generated by Elasticsearch"
                        value={editor.documentID}
                        onChange={(event) => setEditor({ ...editor, documentID: event.target.value })}
                      />
                    </div>
                  ) : editor.mode === 'edit' ? (
                    <div className="form-group">
                      <label>document id</label>
                      <input className="form-control" readOnly value={editor.documentID} />
                    </div>
                  ) : null}
                  <LazyJsonEditor
                    height={520}
                    readOnly={editor.mode === 'view'}
                    value={editor.value}
                    onChange={(value) => setEditor({ ...editor, value })}
                  />
                </div>
                <div className="modal-footer">
                  {editor.mode === 'view' ? (
                    <button
                      className="btn btn-default"
                      disabled={readOnly}
                      title={readOnly ? 'index is read-only' : 'edit document'}
                      type="button"
                      onClick={() => setEditor({ ...editor, mode: 'edit' })}
                    >
                      <Icon name="edit" /> edit
                    </button>
                  ) : (
                    <button className="btn btn-success" disabled={saving} type="button" onClick={() => void saveDocument()}>
                      <Icon name={saving ? 'spinner' : 'save'} spin={saving} /> save
                    </button>
                  )}
                  <button className="btn btn-default" type="button" onClick={() => setEditor(null)}>
                    cancel
                  </button>
                </div>
              </div>
            </div>
          </div>
        </>
      ) : null}
    </>
  );
}

function sourceValue(row?: DataRow) {
  if (!row) return {};
  return row._source ?? row;
}

function DataPager({
  first,
  hasNext,
  hasPrevious,
  last,
  page,
  setPage,
  total,
}: {
  first: number;
  hasNext: boolean;
  hasPrevious: boolean;
  last: number;
  page: number;
  setPage: (page: number) => void;
  total: number;
}) {
  return (
    <div className="inline-flex items-center border border-[#55595c] text-[12px] leading-[1.5]">
      <button
        className="border-r border-[#55595c] bg-transparent px-[10px] py-[4px] text-[#eceeef] disabled:cursor-not-allowed disabled:opacity-40"
        disabled={!hasPrevious}
        type="button"
        onClick={() => setPage(page - 1)}
      >
        Previous
      </button>
      <span className="px-[10px] py-[4px] text-[#eceeef]">
        {formatNumber(first)}-{formatNumber(last)} of {formatNumber(total)}
      </span>
      <button
        className="border-l border-[#55595c] bg-transparent px-[10px] py-[4px] text-[#eceeef] disabled:cursor-not-allowed disabled:opacity-40"
        disabled={!hasNext}
        type="button"
        onClick={() => setPage(page + 1)}
      >
        Next
      </button>
    </div>
  );
}

function querySuggestions(query: string, cursor: number, mode: 'kql' | 'lucene', fields: string[]): QuerySuggestion[] {
  const context = queryCompletionContext(query, cursor);
  const token = context.token.toLowerCase();
  if (context.afterField) {
    return mode === 'kql' ? kqlValueSuggestions : luceneValueSuggestions;
  }
  const cleanFields = fields.filter((field) => field && field !== '_id' && field !== '_score' && field !== '_source');
  const fieldSuggestions = cleanFields
    .filter((field, index, all) => all.indexOf(field) === index)
    .filter((field) => !token || field.toLowerCase().includes(token.replace(/[:<>]=?$/, '')))
    .slice(0, 10)
    .map((field) => ({ detail: 'field', label: field, value: mode === 'kql' ? `${field}: ` : `${field}:` }));
  const operators = mode === 'kql' ? kqlSuggestions : luceneSuggestions;
  const operatorSuggestions = operators.filter((item) => !token || item.label.toLowerCase().startsWith(token)).slice(0, 8);
  if (context.expectField) {
    return [...fieldSuggestions, ...operatorSuggestions].slice(0, 12);
  }
  return [...operatorSuggestions, ...fieldSuggestions].slice(0, 12);
}

const kqlSuggestions: QuerySuggestion[] = [
  { detail: 'operator', label: 'and', value: 'and ' },
  { detail: 'operator', label: 'or', value: 'or ' },
  { detail: 'operator', label: 'not', value: 'not ' },
  { detail: 'operator', label: ':', value: ': ' },
  { detail: 'operator', label: '>=', value: '>= ' },
  { detail: 'operator', label: '<=', value: '<= ' },
  { detail: 'operator', label: 'exists', value: ': *' },
];

const kqlValueSuggestions: QuerySuggestion[] = [
  { detail: 'value', label: '"value"', value: '"value"' },
  { detail: 'value', label: '*', value: '*' },
  { detail: 'operator', label: '>=', value: '>= ' },
  { detail: 'operator', label: '<=', value: '<= ' },
  { detail: 'operator', label: '>', value: '> ' },
  { detail: 'operator', label: '<', value: '< ' },
];

const luceneSuggestions: QuerySuggestion[] = [
  { detail: 'operator', label: 'AND', value: 'AND ' },
  { detail: 'operator', label: 'OR', value: 'OR ' },
  { detail: 'operator', label: 'NOT', value: 'NOT ' },
  { detail: 'operator', label: ':', value: ':' },
  { detail: 'operator', label: '[ TO ]', value: '[ TO ]' },
  { detail: 'operator', label: '*', value: '*' },
];

const luceneValueSuggestions: QuerySuggestion[] = [
  { detail: 'value', label: '"value"', value: '"value"' },
  { detail: 'value', label: '*', value: '*' },
  { detail: 'range', label: '[start TO end]', value: '[ TO ]' },
  { detail: 'range', label: '[start TO *]', value: '[ TO *]' },
];

function currentQueryToken(query: string, cursor: number) {
  const range = queryTokenRange(query, cursor);
  return query.slice(range.start, range.end);
}

function queryCompletionContext(query: string, cursor: number) {
  const range = queryTokenRange(query, cursor);
  const token = query.slice(range.start, range.end);
  const before = query.slice(0, cursor);
  const trimmedBefore = before.trimEnd();
  const afterField = /(?:^|\s|\()[_@A-Za-z0-9][_.@A-Za-z0-9-]*:\s*$/.test(before);
  const expectField = trimmedBefore === '' || /\b(?:and|or|not|AND|OR|NOT)\s*$|\(\s*$/.test(trimmedBefore);
  return { afterField, expectField, token };
}

function queryTokenRange(query: string, cursor: number) {
  let start = cursor;
  let end = cursor;
  while (start > 0 && !/\s|\(|\)/.test(query[start - 1])) start -= 1;
  while (end < query.length && !/\s|\(|\)/.test(query[end])) end += 1;
  return { end, start };
}

function extractMappingFields(raw: unknown, index: string) {
  const root = raw as Record<string, unknown> | undefined;
  const indexMapping = (root?.[index] ?? Object.values(root ?? {})[0]) as { mappings?: unknown } | undefined;
  return flattenMappingProperties(mappingProperties(indexMapping?.mappings)).sort((left, right) => left.localeCompare(right));
}

function mappingProperties(mapping: unknown): Record<string, unknown> | undefined {
  if (!mapping || typeof mapping !== 'object') return undefined;
  const candidate = mapping as { properties?: Record<string, unknown> };
  return candidate.properties;
}

function flattenMappingProperties(properties: Record<string, unknown> | undefined, prefix = ''): string[] {
  if (!properties) return [];
  return Object.entries(properties).flatMap(([name, value]) => {
    const path = prefix ? `${prefix}.${name}` : name;
    const field = value as { fields?: Record<string, unknown>; properties?: Record<string, unknown> } | undefined;
    return [
      path,
      ...flattenMappingProperties(field?.properties, path),
      ...Object.keys(field?.fields ?? {}).map((subfield) => `${path}.${subfield}`),
    ];
  });
}

function indexSettingsReadOnly(raw: unknown, index: string) {
  const root = raw as Record<string, unknown> | undefined;
  const indexSettings = (root?.[index] ?? Object.values(root ?? {})[0]) as Record<string, unknown> | undefined;
  return (
    truthySetting(pathValue(indexSettings, ['settings', 'index', 'blocks', 'write'])) ||
    truthySetting(pathValue(indexSettings, ['settings', 'index', 'blocks', 'read_only'])) ||
    truthySetting(pathValue(indexSettings, ['settings', 'index', 'blocks', 'read_only_allow_delete'])) ||
    truthySetting(pathValue(indexSettings, ['settings', 'index.blocks.write'])) ||
    truthySetting(pathValue(indexSettings, ['settings', 'index.blocks.read_only'])) ||
    truthySetting(pathValue(indexSettings, ['settings', 'index.blocks.read_only_allow_delete']))
  );
}

function pathValue(value: unknown, path: string[]) {
  let current = value;
  for (const key of path) {
    if (!current || typeof current !== 'object') return undefined;
    current = (current as Record<string, unknown>)[key];
  }
  return current;
}

function truthySetting(value: unknown) {
  return value === true || (typeof value === 'string' && value.toLowerCase() === 'true');
}
