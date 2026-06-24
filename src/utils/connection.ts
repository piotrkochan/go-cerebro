import type { HostBodyWritable } from '../api/client';

export const savedHostKey = 'cerebro.currentHost';

export function cleanConnection(value: HostBodyWritable): HostBodyWritable {
  return {
    host: value.host.trim(),
    ...(value.username?.trim() ? { username: value.username.trim() } : {}),
    ...(value.password ? { password: value.password } : {}),
  };
}
