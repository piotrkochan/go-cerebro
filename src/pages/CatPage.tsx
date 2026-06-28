import { useState } from 'react';

import { cat, type HostBodyWritable } from '../api/client';
import { DataTable, type DataTableColumn } from '../components/DataTable';
import { Icon } from '../components/Icon';
import type { Notify } from '../types';
import { errorMessage, textValue } from '../utils/format';

const catApis = [
  'aliases',
  'allocation',
  'count',
  'fielddata',
  'health',
  'indices',
  'master',
  'nodeattrs',
  'nodes',
  'pending tasks',
  'plugins',
  'recovery',
  'repositories',
  'thread pool',
  'shards',
  'segments',
];

type CatRow = Record<string, unknown>;

export function CatPage({ connection, notify }: { connection: HostBodyWritable; notify: Notify }) {
  const [api, setApi] = useState('indices');
  const [rows, setRows] = useState<CatRow[]>([]);
  const [headers, setHeaders] = useState<string[]>([]);
  const [sortCol, setSortCol] = useState('');
  const [sortAsc, setSortAsc] = useState(true);
  const [executed, setExecuted] = useState(false);

  async function run() {
    try {
      const result = await cat<true>({ body: { ...connection, api: api.replace(/ /g, '_') }, throwOnError: true });
      const nextRows = Array.isArray(result.data.data) ? (result.data.data as CatRow[]) : [];
      const nextHeaders = nextRows.length ? Object.keys(nextRows[0]).filter(Boolean) : [];
      setRows(nextRows);
      setHeaders(nextHeaders);
      setSortCol(nextHeaders[0] ?? '');
      setSortAsc(true);
      setExecuted(true);
    } catch (error) {
      notify('danger', `Error executing request: ${errorMessage(error)}`);
    }
  }

  function sort(header: string) {
    if (sortCol === header) {
      setSortAsc((value) => !value);
    } else {
      setSortCol(header);
      setSortAsc(true);
    }
  }

  const sortedRows = [...rows].sort((left, right) => {
    if (!sortCol) return 0;
    const compared = textValue(left[sortCol]).localeCompare(textValue(right[sortCol]), undefined, {
      numeric: true,
      sensitivity: 'base',
    });
    return sortAsc ? compared : -compared;
  });
  const columns: DataTableColumn<CatRow>[] = headers.map((header) => ({
    header: (
      <button className="normal-action border-0 bg-transparent p-0 text-inherit" type="button" onClick={() => sort(header)}>
        {header} {header === sortCol ? <Icon name={sortAsc ? 'caret-down' : 'sort-alpha-desc'} /> : null}
      </button>
    ),
    key: header,
    render: (row) => textValue(row[header]),
  }));

  return (
    <>
      <h4>cat apis</h4>
      <div className="row">
        <div className="col-xs-12">
          <div className="form-group">
            <select className="form-control" value={api} onChange={(event) => setApi(event.target.value)}>
              {catApis.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </select>
          </div>
        </div>
        <div className="col-xs-12 text-right">
          <button className="btn btn-success" type="button" onClick={() => void run()}>
            execute
          </button>
        </div>
      </div>
      <div className="row">
        <div className="col-xs-12">
          {executed && headers.length === 0 && rows.length === 0 ? <h4 className="text-center">No data available</h4> : null}
          {headers.length ? (
            <DataTable columns={columns} getRowKey={(_, index) => String(index)} rows={sortedRows} />
          ) : null}
        </div>
      </div>
    </>
  );
}
