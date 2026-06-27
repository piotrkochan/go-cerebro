import { useState } from 'react';

import type { SettingInput } from '../settingsCatalog';

export function SettingValueInput({
  disabled,
  input,
  onChange,
  value,
}: {
  disabled: boolean;
  input: SettingInput;
  onChange: (value: string) => void;
  value: string;
}) {
  const [customChoiceActive, setCustomChoiceActive] = useState(false);

  if (input.kind === 'select') {
    return (
      <select className="form-control font-mono" disabled={disabled} value={value} onChange={(event) => onChange(event.target.value)}>
        <option value="">not set</option>
        {input.options.map((option) => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
      </select>
    );
  }

  if (input.kind === 'choice-text') {
    const selected = customChoiceActive || (value !== '' && !input.options.includes(value)) ? '__custom' : value;
    return (
      <div className="flex flex-col gap-2 sm:flex-row">
        <select
          className="form-control font-mono sm:w-[170px]"
          disabled={disabled}
          value={selected}
          onChange={(event) => {
            if (event.target.value === '__custom') {
              setCustomChoiceActive(true);
              if (input.options.includes(value)) onChange('');
              return;
            }
            setCustomChoiceActive(false);
            onChange(event.target.value);
          }}
        >
          <option value="">not set</option>
          {input.options.map((option) => (
            <option key={option} value={option}>
              {option}
            </option>
          ))}
          <option value="__custom">custom pattern</option>
        </select>
        {selected === '__custom' ? (
          <input
            className="form-control font-mono"
            disabled={disabled}
            placeholder={input.placeholder ?? ''}
            type="text"
            value={value}
            onChange={(event) => onChange(event.target.value)}
          />
        ) : null}
      </div>
    );
  }

  if (input.kind === 'boolean') {
    return (
      <select className="form-control font-mono" disabled={disabled} value={booleanValue(value)} onChange={(event) => onChange(event.target.value)}>
        <option value="">not set</option>
        <option value="true">true</option>
        <option value="false">false</option>
      </select>
    );
  }

  return (
    <input
      className="form-control font-mono"
      disabled={disabled}
      inputMode={inputMode(input)}
      placeholder={placeholder(input)}
      type="text"
      value={value}
      onChange={(event) => onChange(event.target.value)}
    />
  );
}

function booleanValue(value: string) {
  const normalized = value.trim().toLowerCase();
  if (normalized === 'true' || normalized === 'false') return normalized;
  return '';
}

function inputMode(input: SettingInput) {
  if (input.kind === 'number') return 'decimal';
  return undefined;
}

function placeholder(input: SettingInput) {
  if ('placeholder' in input && input.placeholder) return input.placeholder;
  if (input.kind === 'size') return '512mb, 10gb, 40%';
  if (input.kind === 'time') return '30s, 5m, 1h';
  return '';
}
