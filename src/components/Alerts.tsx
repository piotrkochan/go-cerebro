import type { Alert } from '../types';

const containerClass = 'fixed right-[20px] top-[62px] z-[2000] w-[min(420px,calc(100vw-40px))]';
const baseAlertClass = 'mb-[10px] border px-[12px] py-[10px] opacity-95 shadow-[0_5px_15px_rgba(0,0,0,0.35)]';
const alertKindClass: Record<Alert['kind'], string> = {
  danger: 'border-[#e64759] bg-[#4a252b] text-[#ffb7bf]',
  info: 'border-[#1ca8dd] bg-[#173844] text-[#9bdcf2]',
  success: 'border-[#1ac98e] bg-[#173f34] text-[#9ff0d4]',
};

export function Alerts({ alerts }: { alerts: Alert[] }) {
  return (
    <div className={containerClass}>
      {alerts.map((alert) => (
        <div className={`${baseAlertClass} ${alertKindClass[alert.kind]}`} key={alert.id} role="status">
          {alert.text}
        </div>
      ))}
    </div>
  );
}
