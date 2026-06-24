export function errorMessage(error: unknown): string {
  if (error instanceof Error) return error.message;
  if (typeof error === 'object' && error !== null) {
    const candidate = error as { detail?: unknown; message?: unknown; title?: unknown };
    return textValue(candidate.detail ?? candidate.message ?? candidate.title ?? error);
  }
  return textValue(error);
}

export function parseJson(value: string): unknown {
  if (!value.trim()) return undefined;
  try {
    return JSON.parse(value);
  } catch {
    return value;
  }
}

export function formatJson(value: unknown): string {
  if (value === '') return '';
  return JSON.stringify(value, null, 2);
}

export function textValue(value: unknown): string {
  if (value === null || value === undefined) return '';
  if (typeof value === 'string') return value;
  if (typeof value === 'number' || typeof value === 'boolean') return String(value);
  return JSON.stringify(value);
}

export function numberValue(value: unknown): number {
  if (typeof value === 'number') return value;
  if (typeof value === 'string') {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

export function formatNumber(value: unknown): string {
  return numberValue(value).toLocaleString();
}

export function formatFixed(value: unknown, digits: number): string {
  return numberValue(value).toFixed(digits);
}

export function formatBytes(value: unknown): string {
  const bytes = numberValue(value);
  if (!bytes) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const amount = bytes / 1024 ** exponent;
  return `${amount.toFixed(amount >= 10 || exponent === 0 ? 0 : 1)} ${units[exponent]}`;
}

export function timeInterval(value: unknown): string {
  const ms = numberValue(value);
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${Math.round(ms / 1000)}s`;
  if (ms < 3600000) return `${Math.round(ms / 60000)}m`;
  if (ms < 86400000) return `${Math.round(ms / 3600000)}h`;
  return `${Math.round(ms / 86400000)}d`;
}

export function curl(method: string, path: string, body: string): string {
  const data = body.trim() ? ` -d '${body.trim()}'` : '';
  return `curl -X${method} '${path}'${data}`;
}
