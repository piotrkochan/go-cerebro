import { Fragment, type ReactNode } from 'react';

import { Icon } from './Icon';
import type { SortState } from '../utils/sort';

export type DataTableColumn<T> = {
  className?: string;
  header: ReactNode;
  headerClassName?: string;
  key: string;
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
      {children} {sort.key === name ? <Icon name={sort.order === 'asc' ? 'caret-down' : 'sort-alpha-desc'} /> : null}
    </button>
  );
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
    <table className={`table table-condensed ${className}`.trim()}>
      {showHeader ? (
        <thead>
          <tr>
            {columns.map((column) => (
              <th className={column.headerClassName} key={column.key}>
                {column.header}
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
                    <td className={column.className} key={column.key}>
                      {column.render(row, index)}
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
