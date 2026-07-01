import type { ReactNode } from 'react';

type CheckboxProps = {
  checked: boolean;
  className?: string;
  disabled?: boolean;
  label: ReactNode;
  onChange: (checked: boolean) => void;
};

export function Checkbox({ checked, className = '', disabled = false, label, onChange }: CheckboxProps) {
  return (
    <label className={['m-0 inline-flex cursor-pointer select-none items-center gap-[7px] font-normal', disabled ? 'cursor-not-allowed opacity-60' : '', className].filter(Boolean).join(' ')}>
      <input
        checked={checked}
        className="peer sr-only"
        disabled={disabled}
        type="checkbox"
        onChange={(event) => onChange(event.target.checked)}
      />
      <span className="flex h-[14px] w-[14px] shrink-0 items-center justify-center border border-[#8b8f95] bg-[#2f3234] text-transparent transition-colors peer-checked:border-[#1ca8dd] peer-checked:bg-[#1ca8dd] peer-checked:text-[#f7f7f7] peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-[#8bdbff]">
        <svg aria-hidden="true" className="h-[10px] w-[10px]" fill="none" viewBox="0 0 12 10">
          <path d="M1 5l3 3 7-7" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" />
        </svg>
      </span>
      <span className="min-w-0">{label}</span>
    </label>
  );
}
