import type { ReactNode } from 'react';

import {
  type AllocationRule,
  type BooleanChoice,
  type ILMPhaseWizard,
  type ILMWizardFeatures,
  type ILMWizardValues,
  type MigrateChoice,
  type SamplingMethod,
} from '../forms/ilmPolicyForm';
import { Button } from './Button';
import { Checkbox } from './Checkbox';
import { Icon } from './Icon';

type PhaseName = keyof ILMWizardValues;

export function ILMPolicyWizard({
  features,
  onChange,
  value,
}: {
  features: ILMWizardFeatures;
  onChange: (value: ILMWizardValues) => void;
  value: ILMWizardValues;
}) {
  const changePhase = (name: PhaseName, next: Partial<ILMPhaseWizard>) => {
    onChange({ ...value, [name]: { ...value[name], ...next } });
  };

  return (
    <div className="space-y-[15px]">
      <WizardPhase title="hot phase" enabled={value.hot.enabled} onToggle={(enabled) => changePhase('hot', { enabled })}>
        <SetPriorityAction phase={value.hot} onChange={(setPriority) => changePhase('hot', { setPriority })} />
        <ActionToggle checked={value.hot.unfollow} label="unfollow" onChange={(unfollow) => changePhase('hot', { unfollow })} />
        <RolloverAction phase={value.hot} onChange={(rollover) => changePhase('hot', { rollover })} />
        <ActionToggle checked={value.hot.readOnly} label="read only" onChange={(readOnly) => changePhase('hot', { readOnly })} />
        {features.downsample ? <DownsampleAction phase={value.hot} supportsForceMerge={features.downsampleForceMerge} onChange={(downsample) => changePhase('hot', { downsample })} /> : null}
        <ShrinkAction phase={value.hot} onChange={(shrink) => changePhase('hot', { shrink })} />
        <ForceMergeAction phase={value.hot} onChange={(forceMerge) => changePhase('hot', { forceMerge })} />
        {features.searchableSnapshot ? <SearchableSnapshotAction phase={value.hot} onChange={(searchableSnapshot) => changePhase('hot', { searchableSnapshot })} /> : null}
      </WizardPhase>

      <WizardPhase title="warm phase" enabled={value.warm.enabled} onToggle={(enabled) => changePhase('warm', { enabled })} minAge={value.warm.minAge} onMinAgeChange={(minAge) => changePhase('warm', { minAge })}>
        <SetPriorityAction phase={value.warm} onChange={(setPriority) => changePhase('warm', { setPriority })} />
        <ActionToggle checked={value.warm.unfollow} label="unfollow" onChange={(unfollow) => changePhase('warm', { unfollow })} />
        <ActionToggle checked={value.warm.readOnly} label="read only" onChange={(readOnly) => changePhase('warm', { readOnly })} />
        {features.downsample ? <DownsampleAction phase={value.warm} supportsForceMerge={features.downsampleForceMerge} onChange={(downsample) => changePhase('warm', { downsample })} /> : null}
        <AllocateAction phase={value.warm} onChange={(allocate) => changePhase('warm', { allocate })} />
        {features.migrate ? <MigrateAction value={value.warm.migrate} onChange={(migrate) => changePhase('warm', { migrate })} /> : null}
        <ShrinkAction phase={value.warm} onChange={(shrink) => changePhase('warm', { shrink })} />
        <ForceMergeAction phase={value.warm} onChange={(forceMerge) => changePhase('warm', { forceMerge })} />
      </WizardPhase>

      <WizardPhase title="cold phase" enabled={value.cold.enabled} onToggle={(enabled) => changePhase('cold', { enabled })} minAge={value.cold.minAge} onMinAgeChange={(minAge) => changePhase('cold', { minAge })}>
        <SetPriorityAction phase={value.cold} onChange={(setPriority) => changePhase('cold', { setPriority })} />
        <ActionToggle checked={value.cold.unfollow} label="unfollow" onChange={(unfollow) => changePhase('cold', { unfollow })} />
        <ActionToggle checked={value.cold.readOnly} label="read only" onChange={(readOnly) => changePhase('cold', { readOnly })} />
        {features.downsample ? <DownsampleAction phase={value.cold} supportsForceMerge={features.downsampleForceMerge} onChange={(downsample) => changePhase('cold', { downsample })} /> : null}
        {features.searchableSnapshot ? <SearchableSnapshotAction phase={value.cold} onChange={(searchableSnapshot) => changePhase('cold', { searchableSnapshot })} /> : null}
        <AllocateAction phase={value.cold} onChange={(allocate) => changePhase('cold', { allocate })} />
        {features.migrate ? <MigrateAction value={value.cold.migrate} onChange={(migrate) => changePhase('cold', { migrate })} /> : null}
      </WizardPhase>

      {features.frozenPhase ? (
        <WizardPhase title="frozen phase" enabled={value.frozen.enabled} onToggle={(enabled) => changePhase('frozen', { enabled })} minAge={value.frozen.minAge} onMinAgeChange={(minAge) => changePhase('frozen', { minAge })}>
          <ActionToggle checked={value.frozen.unfollow} label="unfollow" onChange={(unfollow) => changePhase('frozen', { unfollow })} />
          <SearchableSnapshotAction phase={value.frozen} onChange={(searchableSnapshot) => changePhase('frozen', { searchableSnapshot })} />
        </WizardPhase>
      ) : null}

      <WizardPhase title="delete phase" enabled={value.delete.enabled} onToggle={(enabled) => changePhase('delete', { enabled })} minAge={value.delete.minAge} onMinAgeChange={(minAge) => changePhase('delete', { minAge })}>
        {features.waitForSnapshot ? <WaitForSnapshotAction phase={value.delete} onChange={(waitForSnapshot) => changePhase('delete', { waitForSnapshot })} /> : null}
        <ActionToggle checked={value.delete.delete.enabled} label="delete" onChange={(enabled) => changePhase('delete', { delete: { ...value.delete.delete, enabled } })}>
          <BooleanSelect defaultLabel="default (true)" label="delete searchable snapshot" value={value.delete.delete.deleteSearchableSnapshot} onChange={(deleteSearchableSnapshot) => changePhase('delete', { delete: { ...value.delete.delete, deleteSearchableSnapshot } })} />
        </ActionToggle>
      </WizardPhase>
    </div>
  );
}

function RolloverAction({ phase, onChange }: ActionProps<'rollover'>) {
  const value = phase.rollover;
  const change = (next: Partial<typeof value>) => onChange({ ...value, ...next });
  return (
    <ActionToggle checked={value.enabled} label="rollover" onChange={(enabled) => change({ enabled })}>
      <FieldGrid>
        <WizardInput label="max age" placeholder="30d" value={value.maxAge} onChange={(maxAge) => change({ maxAge })} />
        <WizardInput label="max primary shard size" placeholder="50gb" value={value.maxPrimaryShardSize} onChange={(maxPrimaryShardSize) => change({ maxPrimaryShardSize })} />
        <WizardInput label="max index size" placeholder="100gb" value={value.maxSize} onChange={(maxSize) => change({ maxSize })} />
        <WizardInput label="max documents" placeholder="1000000" value={value.maxDocs} onChange={(maxDocs) => change({ maxDocs })} />
        <WizardInput label="max primary shard documents" placeholder="10000000" value={value.maxPrimaryShardDocs} onChange={(maxPrimaryShardDocs) => change({ maxPrimaryShardDocs })} />
        <WizardInput label="min age" placeholder="1d" value={value.minAge} onChange={(minAge) => change({ minAge })} />
        <WizardInput label="min primary shard size" placeholder="1gb" value={value.minPrimaryShardSize} onChange={(minPrimaryShardSize) => change({ minPrimaryShardSize })} />
        <WizardInput label="min index size" placeholder="10gb" value={value.minSize} onChange={(minSize) => change({ minSize })} />
        <WizardInput label="min documents" placeholder="1000" value={value.minDocs} onChange={(minDocs) => change({ minDocs })} />
        <WizardInput label="min primary shard documents" placeholder="1000" value={value.minPrimaryShardDocs} onChange={(minPrimaryShardDocs) => change({ minPrimaryShardDocs })} />
      </FieldGrid>
    </ActionToggle>
  );
}

function SetPriorityAction({ phase, onChange }: ActionProps<'setPriority'>) {
  const value = phase.setPriority;
  return (
    <ActionToggle checked={value.enabled} label="set priority" onChange={(enabled) => onChange({ ...value, enabled })}>
      <FieldGrid>
        <WizardInput label="priority" placeholder="100" value={value.priority} onChange={(priority) => onChange({ ...value, priority })} />
      </FieldGrid>
    </ActionToggle>
  );
}

function DownsampleAction({ phase, onChange, supportsForceMerge }: ActionProps<'downsample'> & { supportsForceMerge: boolean }) {
  const value = phase.downsample;
  return (
    <ActionToggle checked={value.enabled} label="downsample" onChange={(enabled) => onChange({ ...value, enabled })}>
      <FieldGrid>
        <WizardInput label="fixed interval" placeholder="1h" value={value.fixedInterval} onChange={(fixedInterval) => onChange({ ...value, fixedInterval })} />
        {supportsForceMerge ? (
          <>
            <BooleanSelect defaultLabel="default (true)" label="force merge index" value={value.forceMergeIndex} onChange={(forceMergeIndex) => onChange({ ...value, forceMergeIndex })} />
            <WizardSelect
              label="sampling method"
              value={value.samplingMethod}
              options={[
                ['default', 'default (aggregate)'],
                ['aggregate', 'aggregate'],
                ['last_value', 'last value'],
              ]}
              onChange={(samplingMethod) => onChange({ ...value, samplingMethod: samplingMethod as SamplingMethod })}
            />
          </>
        ) : null}
      </FieldGrid>
    </ActionToggle>
  );
}

function ShrinkAction({ phase, onChange }: ActionProps<'shrink'>) {
  const value = phase.shrink;
  return (
    <ActionToggle checked={value.enabled} label="shrink" onChange={(enabled) => onChange({ ...value, enabled })}>
      <FieldGrid>
        <WizardInput label="number of shards" placeholder="1" value={value.numberOfShards} onChange={(numberOfShards) => onChange({ ...value, numberOfShards, maxPrimaryShardSize: numberOfShards ? '' : value.maxPrimaryShardSize })} />
        <WizardInput label="or max primary shard size" placeholder="50gb" value={value.maxPrimaryShardSize} onChange={(maxPrimaryShardSize) => onChange({ ...value, maxPrimaryShardSize, numberOfShards: maxPrimaryShardSize ? '' : value.numberOfShards })} />
      </FieldGrid>
    </ActionToggle>
  );
}

function ForceMergeAction({ phase, onChange }: ActionProps<'forceMerge'>) {
  const value = phase.forceMerge;
  return (
    <ActionToggle checked={value.enabled} label="force merge" onChange={(enabled) => onChange({ ...value, enabled })}>
      <FieldGrid>
        <WizardInput label="max segments" placeholder="1" value={value.maxNumSegments} onChange={(maxNumSegments) => onChange({ ...value, maxNumSegments })} />
      </FieldGrid>
    </ActionToggle>
  );
}

function AllocateAction({ phase, onChange }: ActionProps<'allocate'>) {
  const value = phase.allocate;
  return (
    <ActionToggle checked={value.enabled} label="allocate" onChange={(enabled) => onChange({ ...value, enabled })}>
      <FieldGrid>
        <WizardInput label="number of replicas" placeholder="1" value={value.numberOfReplicas} onChange={(numberOfReplicas) => onChange({ ...value, numberOfReplicas })} />
        <WizardInput label="total shards per node" placeholder="-1" value={value.totalShardsPerNode} onChange={(totalShardsPerNode) => onChange({ ...value, totalShardsPerNode })} />
      </FieldGrid>
      <AllocationRules label="include" rules={value.include} onChange={(include) => onChange({ ...value, include })} />
      <AllocationRules label="exclude" rules={value.exclude} onChange={(exclude) => onChange({ ...value, exclude })} />
      <AllocationRules label="require" rules={value.require} onChange={(require) => onChange({ ...value, require })} />
    </ActionToggle>
  );
}

function MigrateAction({ value, onChange }: { value: MigrateChoice; onChange: (value: MigrateChoice) => void }) {
  return (
    <div className="border-t border-[#4b4f51] py-[9px] first:border-t-0">
      <WizardSelect
        label="data tier migration"
        value={value}
        options={[
          ['automatic', 'automatic (default)'],
          ['enabled', 'explicitly enabled'],
          ['disabled', 'disabled'],
        ]}
        onChange={(next) => onChange(next as MigrateChoice)}
      />
    </div>
  );
}

function SearchableSnapshotAction({ phase, onChange }: ActionProps<'searchableSnapshot'>) {
  const value = phase.searchableSnapshot;
  return (
    <ActionToggle checked={value.enabled} label="searchable snapshot" onChange={(enabled) => onChange({ ...value, enabled })}>
      <FieldGrid>
        <WizardInput label="snapshot repository" placeholder="repository-name" value={value.snapshotRepository} onChange={(snapshotRepository) => onChange({ ...value, snapshotRepository })} />
        <WizardInput label="replicate for" placeholder="14d" value={value.replicateFor} onChange={(replicateFor) => onChange({ ...value, replicateFor })} />
        <WizardInput label="total shards per node" placeholder="unlimited" value={value.totalShardsPerNode} onChange={(totalShardsPerNode) => onChange({ ...value, totalShardsPerNode })} />
        <BooleanSelect defaultLabel="default (true)" label="force merge index" value={value.forceMergeIndex} onChange={(forceMergeIndex) => onChange({ ...value, forceMergeIndex })} />
      </FieldGrid>
    </ActionToggle>
  );
}

function WaitForSnapshotAction({ phase, onChange }: ActionProps<'waitForSnapshot'>) {
  const value = phase.waitForSnapshot;
  return (
    <ActionToggle checked={value.enabled} label="wait for snapshot" onChange={(enabled) => onChange({ ...value, enabled })}>
      <FieldGrid>
        <WizardInput label="SLM policy" placeholder="policy-name" value={value.policy} onChange={(policy) => onChange({ ...value, policy })} />
      </FieldGrid>
    </ActionToggle>
  );
}

type ActionKey = 'allocate' | 'downsample' | 'forceMerge' | 'rollover' | 'searchableSnapshot' | 'setPriority' | 'shrink' | 'waitForSnapshot';
type ActionProps<K extends ActionKey> = {
  phase: ILMPhaseWizard;
  onChange: (value: ILMPhaseWizard[K]) => void;
};

function WizardPhase({
  children,
  enabled,
  minAge,
  onMinAgeChange,
  onToggle,
  title,
}: {
  children: ReactNode;
  enabled: boolean;
  minAge?: string;
  onMinAgeChange?: (value: string) => void;
  onToggle: (enabled: boolean) => void;
  title: string;
}) {
  return (
    <section className="border border-[#55595c] p-[12px]">
      <div className="flex flex-wrap items-center justify-between gap-[10px]">
        <h4 className="!m-0">{title}</h4>
        <Checkbox checked={enabled} label="enabled" onChange={onToggle} />
      </div>
      {enabled ? (
        <div className="mt-[10px]">
          {onMinAgeChange ? (
            <div className="mb-[8px] max-w-[280px]">
              <WizardInput compact label="minimum age" placeholder="0d" value={minAge ?? ''} onChange={onMinAgeChange} />
            </div>
          ) : null}
          {children}
        </div>
      ) : <div className="mt-[10px] info-text">disabled</div>}
    </section>
  );
}

function ActionToggle({
  checked,
  children,
  label,
  onChange,
}: {
  checked: boolean;
  children?: ReactNode;
  label: string;
  onChange: (value: boolean) => void;
}) {
  return (
    <div className="border-t border-[#4b4f51] py-[9px] first:border-t-0">
      <Checkbox checked={checked} label={label} onChange={onChange} />
      {checked && children ? <div className="ml-[7px] mt-[9px] border-l-2 border-[#55595c] pl-[12px]">{children}</div> : null}
    </div>
  );
}

function FieldGrid({ children }: { children: ReactNode }) {
  return <div className="grid grid-cols-[repeat(auto-fit,minmax(210px,1fr))] gap-x-[15px]">{children}</div>;
}

function WizardInput({
  compact = false,
  label,
  onChange,
  placeholder,
  value,
}: {
  compact?: boolean;
  label: string;
  onChange: (value: string) => void;
  placeholder: string;
  value: string;
}) {
  return (
    <label className={compact ? 'block' : 'form-group block'}>
      <span className="form-label block">{label}</span>
      <input className="form-control font-mono" placeholder={placeholder} value={value} onChange={(event) => onChange(event.target.value)} />
    </label>
  );
}

function BooleanSelect({
  defaultLabel,
  label,
  onChange,
  value,
}: {
  defaultLabel: string;
  label: string;
  onChange: (value: BooleanChoice) => void;
  value: BooleanChoice;
}) {
  return (
    <WizardSelect
      label={label}
      value={value}
      options={[
        ['default', defaultLabel],
        ['true', 'true'],
        ['false', 'false'],
      ]}
      onChange={(next) => onChange(next as BooleanChoice)}
    />
  );
}

function WizardSelect({
  label,
  onChange,
  options,
  value,
}: {
  label: string;
  onChange: (value: string) => void;
  options: Array<[string, string]>;
  value: string;
}) {
  return (
    <label className="form-group block">
      <span className="form-label block">{label}</span>
      <select className="form-control" value={value} onChange={(event) => onChange(event.target.value)}>
        {options.map(([optionValue, optionLabel]) => <option key={optionValue} value={optionValue}>{optionLabel}</option>)}
      </select>
    </label>
  );
}

function AllocationRules({
  label,
  onChange,
  rules,
}: {
  label: string;
  onChange: (rules: AllocationRule[]) => void;
  rules: AllocationRule[];
}) {
  return (
    <div className="mb-[10px] last:mb-0">
      <div className="mb-[6px] flex items-center justify-between gap-[10px]">
        <span className="form-label">{label} rules</span>
        <Button icon="plus" size="xs" onClick={() => onChange([...rules, { attribute: '', value: '' }])}>add rule</Button>
      </div>
      {rules.length ? (
        <div className="space-y-[6px]">
          {rules.map((rule, index) => (
            <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_28px] gap-[6px]" key={`${label}-${index}`}>
              <input
                aria-label={`${label} attribute`}
                className="form-control font-mono"
                placeholder="attribute"
                value={rule.attribute}
                onChange={(event) => onChange(updateRule(rules, index, { attribute: event.target.value }))}
              />
              <input
                aria-label={`${label} values`}
                className="form-control font-mono"
                placeholder="value1,value2"
                value={rule.value}
                onChange={(event) => onChange(updateRule(rules, index, { value: event.target.value }))}
              />
              <button className="btn btn-danger btn-xs !px-0" title={`remove ${label} rule`} type="button" onClick={() => onChange(rules.filter((_, ruleIndex) => ruleIndex !== index))}>
                <Icon name="trash" />
              </button>
            </div>
          ))}
        </div>
      ) : <div className="info-text">no {label} rules</div>}
    </div>
  );
}

function updateRule(rules: AllocationRule[], index: number, next: Partial<AllocationRule>) {
  return rules.map((rule, ruleIndex) => ruleIndex === index ? { ...rule, ...next } : rule);
}
