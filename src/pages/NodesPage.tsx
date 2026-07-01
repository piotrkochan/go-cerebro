import { useEffect, useState } from 'react';

import { nodes, type HostBodyWritable, type Node } from '../api/client';
import { Checkbox } from '../components/Checkbox';
import { Icon } from '../components/Icon';
import { NodeCell, SortLink, compareNodes, roleIcon } from '../components/LegacyUi';
import { clusterPath } from '../utils/connection';
import { formatBytes, formatFixed, formatNumber, textValue, timeInterval } from '../utils/format';

type NodeSort = keyof Node | 'cpu.load' | 'cpu.process' | 'heap.percent' | 'disk.percent';

const inputClass =
  'block h-[34px] w-full border border-[#55595c] bg-[#434749] px-3 py-1.5 text-[13px] leading-normal text-[#eceeef] outline-none placeholder:text-[#8b8f95] focus:border-[#1ca8dd]';
const tableClass =
  'mb-[19px] w-full max-w-full border border-[#6f7579] bg-transparent [&_td]:border [&_td]:border-[#6f7579] [&_td]:p-2 [&_td]:align-top [&_th]:border [&_th]:border-[#6f7579] [&_th]:p-2 [&_th]:text-left';
const mainStatClass = 'float-left mr-[15px] min-w-[70px] text-[36px] font-light leading-[40px]';
const detailStatClass = 'float-left leading-5';
const emptyStatClass = 'mr-[15px] min-w-[70px] text-center text-[36px] font-light leading-[40px]';

export function NodesPage({ connection, refreshTick }: { connection: HostBodyWritable; refreshTick: number }) {
  const [data, setData] = useState<Node[]>([]);
  const [filter, setFilter] = useState({ coordinating: false, data: false, ingest: false, master: false, name: '' });
  const [sortBy, setSortBy] = useState<NodeSort>('name');
  const [reverse, setReverse] = useState(false);

  useEffect(() => {
    let ignore = false;
    async function load() {
      const result = await nodes<true>({ path: clusterPath(connection), throwOnError: true });
      if (!ignore) setData(result.data.items ?? []);
    }
    void load();
    return () => {
      ignore = true;
    };
  }, [connection, refreshTick]);

  const filtered = data
    .filter((node) => textValue(node.name).toLowerCase().includes(filter.name.toLowerCase()))
    .filter((node) => !filter.master || node.master)
    .filter((node) => !filter.data || node.data)
    .filter((node) => !filter.ingest || node.ingest)
    .filter((node) => !filter.coordinating || node.coordinating)
    .sort((left, right) => compareNodes(left, right, sortBy, reverse));

  function sort(property: NodeSort) {
    if (sortBy === property) {
      setReverse((value) => !value);
    } else {
      setSortBy(property);
      setReverse(false);
    }
  }

  return (
    <>
      <div className="mb-[15px] flex flex-wrap items-center gap-x-[30px] gap-y-[10px]">
        <div className="w-full sm:w-[260px] lg:w-1/4">
          <input
            className={inputClass}
            placeholder="filter nodes by name"
            type="text"
            value={filter.name}
            onChange={(event) => setFilter((value) => ({ ...value, name: event.target.value }))}
          />
        </div>
        <div className="flex flex-wrap items-center gap-x-6 gap-y-2 select-none">
          {(['master', 'data', 'ingest', 'coordinating'] as const).map((role) => (
            <Checkbox
              checked={filter[role]}
              key={role}
              label={<><Icon name={roleIcon(role)} /> {role}</>}
              onChange={(checked) => setFilter((value) => ({ ...value, [role]: checked }))}
            />
          ))}
        </div>
      </div>
      <table className={tableClass}>
        <thead>
          <tr>
            <th><SortLink active={sortBy === 'name'} reverse={reverse} text="name" onClick={() => sort('name')} /></th>
            <th><SortLink active={sortBy === 'cpu.load'} reverse={reverse} text="load" onClick={() => sort('cpu.load')} /></th>
            <th><SortLink active={sortBy === 'cpu.process'} reverse={reverse} text="process cpu %" onClick={() => sort('cpu.process')} /></th>
            <th><SortLink active={sortBy === 'heap.percent'} reverse={reverse} text="heap usage %" onClick={() => sort('heap.percent')} /></th>
            <th><SortLink active={sortBy === 'disk.percent'} reverse={reverse} text="disk usage %" onClick={() => sort('disk.percent')} /></th>
            <th><SortLink active={sortBy === 'uptime'} reverse={reverse} text="uptime" onClick={() => sort('uptime')} /></th>
          </tr>
        </thead>
        <tbody>
          {filtered.map((node) => (
            <tr key={node.id}>
              <td><NodeCell node={node} /></td>
              <td><span className={mainStatClass}>{formatFixed(node.cpu?.load, 2)}</span></td>
              <td>
                <span className={mainStatClass}>{formatNumber(node.cpu?.process)}%</span>
                <span className={detailStatClass}><div>os cpu: {formatNumber(node.cpu?.os)}%</div></span>
              </td>
              <td>
                <span className={mainStatClass}>{formatNumber(node.heap?.percent)}%</span>
                <span className={detailStatClass}><div>used: {textValue(node.heap?.used)}</div><div>max: {textValue(node.heap?.max)}</div></span>
              </td>
              <td>
                {node.disk ? (
                  <>
                    <span className={mainStatClass}>{formatNumber(node.disk.percent)}%</span>
                    <span className={detailStatClass}>
                      <div>available: {formatBytes(node.disk.available)}</div>
                      <div>total: {formatBytes(node.disk.total)}</div>
                    </span>
                  </>
                ) : <div className={emptyStatClass}>-</div>}
              </td>
              <td><span className={mainStatClass}>{timeInterval(node.uptime)}</span></td>
            </tr>
          ))}
        </tbody>
      </table>
    </>
  );
}
