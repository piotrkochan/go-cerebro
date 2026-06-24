export type SortOrder = 'asc' | 'desc';

export type SortState<Key extends string> = {
  key: Key;
  order: SortOrder;
};

export function nextSort<Key extends string>(current: SortState<Key>, key: Key): SortState<Key> {
  return {
    key,
    order: current.key === key && current.order === 'asc' ? 'desc' : 'asc',
  };
}

export function sortByText<Item, Key extends string>(
  items: Item[],
  sort: SortState<Key>,
  value: (item: Item, key: Key) => string,
) {
  return [...items].sort((left, right) => compareText(value(left, sort.key), value(right, sort.key), sort.order));
}

export function compareText(left: string, right: string, order: SortOrder) {
  const result = left.localeCompare(right, undefined, { numeric: true, sensitivity: 'base' });
  return order === 'asc' ? result : -result;
}
