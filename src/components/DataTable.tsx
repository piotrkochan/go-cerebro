import { Fragment, type ReactNode } from 'react';

import { Icon } from './Icon';
import type { SortState } from '../utils/sort';

export type DataTableColumn<T> = {
  className?: string;
  header: ReactNode;
  headerClassName?: string;
  key: string;
  width?: string;
  render: (row: T, index: number) => ReactNode;
};

export function SortHeader<K extends string>({
  children,
  name,
  onSort,
  sort,
}: {
  children: ReactNode;
  name: K;
  onSort: (name: K) => void;
  sort: SortState<K>;
}) {
  return (
    <button className="normal-action border-0 bg-transparent p-0 text-inherit" type="button" onClick={() => onSort(name)}>
      {children} <SortIndicator active={sort.key === name} order={sort.order} />
    </button>
  );
}

export function SortIndicator({ active, order }: { active: boolean; order: 'asc' | 'desc' }) {
  if (!active) return null;
  return <Icon className="ml-[3px] inline-block align-[-2px]" name={order === 'asc' ? 'sort-alpha-asc' : 'sort-alpha-desc'} size={13} />;
}

export function DataTable<T>({
  className = '',
  columns,
  empty = 'none',
  getRowKey,
  onRowClick,
  renderDetail,
  rowClassName,
  rows,
  showHeader = true,
}: {
  className?: string;
  columns: DataTableColumn<T>[];
  empty?: ReactNode;
  getRowKey: (row: T, index: number) => string;
  onRowClick?: (row: T, index: number) => void;
  renderDetail?: (row: T, index: number) => ReactNode;
  rowClassName?: string | ((row: T, index: number) => string);
  rows: T[];
  showHeader?: boolean;
}) {
  return (
    <table className={`table table-condensed table-fixed ${className}`.trim()}>
      <colgroup>
        {columns.map((column) => (
          <col key={column.key} style={column.width ? { width: column.width } : column.key === 'actions' ? { width: '116px' } : undefined} />
        ))}
      </colgroup>
      {showHeader ? (
        <thead className="bg-[#343739]">
          <tr>
            {columns.map((column) => (
              <th className={['border-b border-[#55595c]', column.key === 'actions' ? 'w-[116px]' : '', column.headerClassName].filter(Boolean).join(' ')} key={column.key} title={typeof column.header === 'string' ? column.header : undefined}>
                <div className="min-w-0 truncate">{column.header}</div>
              </th>
            ))}
          </tr>
        </thead>
      ) : null}
      <tbody>
        {rows.length ? (
          rows.map((row, index) => {
            const key = getRowKey(row, index);
            const detail = renderDetail?.(row, index);
            const trClassName = typeof rowClassName === 'function' ? rowClassName(row, index) : rowClassName;
            return (
              <Fragment key={key}>
                <tr className={trClassName} onClick={onRowClick ? () => onRowClick(row, index) : undefined}>
                  {columns.map((column) => (
                    <td className={[column.key === 'actions' ? 'whitespace-nowrap' : '', column.className].filter(Boolean).join(' ')} key={column.key}>
                      <div className={column.key === 'actions' ? 'min-w-0 overflow-visible' : 'min-w-0 truncate'}>{column.render(row, index)}</div>
                    </td>
                  ))}
                </tr>
                {detail ? (
                  <tr>
                    <td className="info-text" colSpan={columns.length}>
                      {detail}
                    </td>
                  </tr>
                ) : null}
              </Fragment>
            );
          })
        ) : (
          <tr>
            <td className="info-text" colSpan={columns.length}>
              {empty}
            </td>
          </tr>
        )}
      </tbody>
    </table>
  );
}
