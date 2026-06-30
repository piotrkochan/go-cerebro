import type { HostBodyWritable } from '../api/client';

export const savedHostKey = 'cerebro.currentHost';
export const savedHostNameKey = 'cerebro.currentHostName';

export function cleanConnection(value: HostBodyWritable): HostBodyWritable {
  return {
    host: value.host.trim(),
  };
}

export function clusterPath(value: HostBodyWritable): { cluster: string } {
  return { cluster: cleanConnection(value).host };
}
