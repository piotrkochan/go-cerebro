import { lazy, Suspense } from 'react';

const JsonEditor = lazy(() => import('./JsonEditor').then((module) => ({ default: module.JsonEditor })));

export function LazyJsonEditor({
  height,
  onChange,
  readOnly = false,
  value,
}: {
  height: number;
  onChange: (value: string) => void;
  readOnly?: boolean;
  value: string;
}) {
  return (
    <Suspense fallback={<div style={{ border: '1px solid #55595c', height }} />}>
      <JsonEditor height={height} readOnly={readOnly} value={value} onChange={onChange} />
    </Suspense>
  );
}
