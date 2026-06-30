import { useEffect, useState } from 'react';
import { useForm } from '@tanstack/react-form';

import { restHistory, restIndex, restRequest, type HostBodyWritable } from '../api/client';
import { DataTable, SortHeader, type DataTableColumn } from '../components/DataTable';
import { Icon } from '../components/Icon';
import { LazyJsonEditor } from '../components/LazyJsonEditor';
import { SplitPane } from '../components/SplitPane';
import { restRequestFormDefaults, type RestRequestFormValues } from '../forms/restRequestForm';
import { clusterPath } from '../utils/connection';
import { curl, formatJson, parseJson, textValue } from '../utils/format';
import { nextSort, sortByText, type SortState } from '../utils/sort';

const restSplitKey = 'cerebro.restSplitPercent';

type RestResponseState = {
  body: unknown;
  durationMs: number;
  status: number;
};
type RestHistoryItem = { body?: unknown; created_at?: unknown; method?: unknown; path?: unknown };
type RestHistorySortKey = 'created_at' | 'method' | 'path';

export function RestPage({ connection }: { connection: HostBodyWritable }) {
  const [curlCopied, setCurlCopied] = useState(false);
  const [executing, setExecuting] = useState(false);
  const [pathFocused, setPathFocused] = useState(false);
  const [response, setResponse] = useState<RestResponseState>();
  const [history, setHistory] = useState<RestHistoryItem[]>([]);
  const [historySort, setHistorySort] = useState<SortState<RestHistorySortKey>>({ key: 'created_at', order: 'desc' });
  const [pathSuggestions, setPathSuggestions] = useState<string[]>([]);
  const [showHistory, setShowHistory] = useState(false);
  const form = useForm({
    defaultValues: restRequestFormDefaults,
    onSubmit: async ({ value }) => {
      await execute(value);
    },
  });

  useEffect(() => {
    void loadHistory();
    void loadPathSuggestions();
  }, [connection]);

  async function execute(values: RestRequestFormValues) {
    setExecuting(true);
    const startedAt = performance.now();
    try {
      const result = await restRequest<true>({
        body: { data: parseJson(values.body), method: values.method, path: values.path },
        path: clusterPath(connection),
        throwOnError: true,
      });
      setResponse({ body: result.data.data, durationMs: Math.round(performance.now() - startedAt), status: result.data.status });
      await loadHistory();
    } catch (error) {
      setResponse({ body: restErrorBody(error), durationMs: Math.round(performance.now() - startedAt), status: 0 });
    } finally {
      setExecuting(false);
    }
  }

  async function loadHistory() {
    try {
      const result = await restHistory<true>({ path: clusterPath(connection), throwOnError: true });
      setHistory(Array.isArray(result.data.data) ? result.data.data : []);
    } catch {
      setHistory([]);
    }
  }

  async function loadPathSuggestions() {
    const common = ['_cluster/health', '_cat/indices?format=json', '_nodes', '_nodes/stats', '_aliases', '_mapping'];
    try {
      const result = await restIndex<true>({ path: clusterPath(connection), throwOnError: true });
      const data = result.data.data as { indices?: unknown[] } | undefined;
      const indices = Array.isArray(data?.indices) ? data.indices.map(textValue).filter(Boolean) : [];
      setPathSuggestions([
        ...common,
        ...indices.flatMap((index) => [`${index}/_search`, `${index}/_mapping`, `${index}/_settings`, `${index}/_stats`]),
      ]);
    } catch {
      setPathSuggestions(common);
    }
  }

  function setRequest(values: RestRequestFormValues) {
    form.setFieldValue('method', values.method);
    form.setFieldValue('path', values.path);
    form.setFieldValue('body', values.body);
  }

  async function copyCurl(values: RestRequestFormValues) {
    await copyText(curl(values.method, values.path, values.body));
    setCurlCopied(true);
    window.setTimeout(() => setCurlCopied(false), 1200);
  }

  const sortedHistory = sortByText(history, historySort, restHistorySortValue);
  const historyColumns: DataTableColumn<RestHistoryItem>[] = [
    {
      className: 'normal-action',
      header: <SortHeader name="created_at" sort={historySort} onSort={(name) => setHistorySort((value) => nextSort(value, name))}>created</SortHeader>,
      headerClassName: 'w-[100px]',
      key: 'created_at',
      render: (item) => textValue(item.created_at),
    },
    {
      className: 'normal-action',
      header: <SortHeader name="method" sort={historySort} onSort={(name) => setHistorySort((value) => nextSort(value, name))}>method</SortHeader>,
      headerClassName: 'w-[60px]',
      key: 'method',
      render: (item) => textValue(item.method),
    },
    {
      className: 'normal-action',
      header: <SortHeader name="path" sort={historySort} onSort={(name) => setHistorySort((value) => nextSort(value, name))}>path</SortHeader>,
      key: 'path',
      render: (item) => textValue(item.path),
    },
  ];

  return (
    <SplitPane
      storageKey={restSplitKey}
      left={
        <>
        <div className="row">
          <div className="col-lg-10 col-md-9 typeahead-demo form-group">
            <form.Field name="path">
              {(field) => {
                const matches = pathSuggestions
                  .filter((path) => path.toLowerCase().includes(field.state.value.toLowerCase()))
                  .slice(0, 8);
                return (
                  <div className="relative">
                    <input
                      className="form-control form-control-sm"
                      placeholder="path (eg: indexName/_search)"
                      type="text"
                      value={field.state.value}
                      onBlur={() => window.setTimeout(() => setPathFocused(false), 120)}
                      onChange={(event) => field.handleChange(event.target.value)}
                      onFocus={() => setPathFocused(true)}
                    />
                    {pathFocused && matches.length ? (
                      <div className="absolute left-0 right-0 top-full z-[1000] mt-1 max-h-56 overflow-auto border border-[#55595c] bg-[#373a3c] shadow-lg">
                        {matches.map((path) => (
                          <button
                            className="block w-full cursor-pointer truncate px-3 py-1.5 text-left text-[#eceeef] hover:bg-[#434749] hover:text-white"
                            key={path}
                            type="button"
                            onMouseDown={(event) => {
                              event.preventDefault();
                              field.handleChange(path);
                              setPathFocused(false);
                            }}
                          >
                            {path}
                          </button>
                        ))}
                      </div>
                    ) : null}
                  </div>
                );
              }}
            </form.Field>
          </div>
          <div className="col-lg-2 col-md-3 form-group">
            <form.Field name="method">
              {(field) => (
                <select className="form-control" value={field.state.value} onChange={(event) => field.handleChange(event.target.value)}>
                  {['GET', 'PUT', 'POST', 'DELETE'].map((value) => <option key={value}>{value}</option>)}
                </select>
              )}
            </form.Field>
          </div>
        </div>
        <div className="row">
          <div className="col-xs-12">
            <a href="#restHistory" onClick={(event) => {
              event.preventDefault();
              setShowHistory((value) => !value);
            }}><Icon name="history" /> previous requests</a>
          </div>
          {showHistory && history.length ? (
            <div className="col-xs-12 panel-collapse collapse in !block !visible" id="restHistory">
              <DataTable
                columns={historyColumns}
                getRowKey={(_, index) => String(index)}
                rowClassName="normal-action"
                rows={sortedHistory}
                onRowClick={(item) => {
                  setRequest({
                    body: textValue(item.body),
                    method: textValue(item.method) || 'GET',
                    path: textValue(item.path),
                  });
                }}
              />
            </div>
          ) : null}
        </div>
        <div className="form-group row">
          <div className="col-lg-12">
            <form.Field name="body">
              {(field) => <LazyJsonEditor height={600} value={field.state.value} onChange={field.handleChange} />}
            </form.Field>
          </div>
        </div>
        <div className="form-group row">
          <div className="col-lg-12 text-right">
            <div className="btn-group">
              <form.Subscribe selector={(state) => state.values}>
                {(values) => (
                  <button className="btn btn-default" type="button" onClick={() => void copyCurl(values)}>
                    <Icon name="clipboard" /> {curlCopied ? 'copied' : 'cURL'}
                  </button>
                )}
              </form.Subscribe>
            </div>
            <div className="btn-group">
              <form.Subscribe selector={(state) => state.values.body}>
                {(body) => (
                  <button className="btn btn-default" type="button" onClick={() => form.setFieldValue('body', formatJson(parseJson(body) ?? {}))}>
                    <Icon name="align-left" /> format
                  </button>
                )}
              </form.Subscribe>
            </div>
            <div className="btn-group">
              <button className="btn btn-success" disabled={executing} type="button" onClick={() => void form.handleSubmit()}>
                <Icon name={executing ? 'spinner' : 'bolt'} spin={executing} /> send
              </button>
            </div>
          </div>
        </div>
        </>
      }
      right={
        <div style={{ border: '1px solid #55595c', minHeight: 647, overflow: 'hidden', display: 'block' }}>
          {response ? (
            <div className="flex min-h-10 items-center justify-between border-b border-[#55595c] px-[15px] py-[7px]">
              <div>
                <span className={`label ${response.status >= 200 && response.status < 300 ? 'label-success' : 'label-danger'}`}>
                  {response.status || 'error'}
                </span>
                <span className="info-text"> {response.durationMs}ms</span>
              </div>
              <button className="btn btn-default btn-xs" type="button" onClick={() => void copyText(formatJson(response.body))}>
                <Icon name="clipboard" /> copy
              </button>
            </div>
          ) : null}
          <LazyJsonEditor height={response ? 606 : 647} readOnly value={formatJson(restResponseBody(response?.body ?? ''))} onChange={() => undefined} />
        </div>
      }
    />
  );
}

async function copyText(value: string) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return;
    } catch {
      // Fall back to a temporary textarea when the browser blocks Clipboard API.
    }
  }
  const textarea = document.createElement('textarea');
  textarea.value = value;
  textarea.setAttribute('readonly', 'true');
  textarea.style.position = 'fixed';
  textarea.style.opacity = '0';
  document.body.appendChild(textarea);
  textarea.select();
  document.execCommand('copy');
  textarea.remove();
}

function restHistorySortValue(item: RestHistoryItem, key: RestHistorySortKey) {
  return textValue(item[key]);
}

function restErrorBody(error: unknown): unknown {
  if (typeof error === 'object' && error !== null) {
    const candidate = error as { data?: unknown; detail?: unknown; error?: unknown; message?: unknown; status?: unknown; title?: unknown };
    if (candidate.data !== undefined) return candidate.data;
    return {
      error: candidate.error ?? candidate.detail ?? candidate.message ?? candidate.title ?? error,
      status: candidate.status,
    };
  }
  return { error };
}

function restResponseBody(body: unknown): unknown {
  if (typeof body !== 'string') return body;
  const parsed = parseJson(body);
  return typeof parsed === 'string' ? body : parsed;
}
