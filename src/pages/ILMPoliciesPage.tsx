import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { Link } from '@tanstack/react-router';
import { useForm } from '@tanstack/react-form';

import { ilmPoliciesDelete, ilmPoliciesList, ilmPoliciesSave } from '../api/ilmClient';
import type { HostBodyWritable, IlmPolicy } from '../api/client/types.gen';
import { Button } from '../components/Button';
import { Checkbox } from '../components/Checkbox';
import { DataTable, SortIndicator, type DataTableColumn } from '../components/DataTable';
import { Icon } from '../components/Icon';
import { LazyJsonEditor } from '../components/LazyJsonEditor';
import { ConfirmModal } from '../components/Modal';
import { SplitPane } from '../components/SplitPane';
import type { Notify } from '../types';
import { clusterPath } from '../utils/connection';
import { errorMessage, formatJson, parseJson, textValue } from '../utils/format';
import { nextSort, sortByText, type SortState } from '../utils/sort';

type ILMSortKey = 'name' | 'phases' | 'version';
type PolicyEditorMode = 'wizard' | 'json';

type ILMWizardValues = {
  hotRollover: boolean;
  hotMaxAge: string;
  hotMaxDocs: string;
  hotMaxPrimaryShardSize: string;
  hotMaxSize: string;
  warmEnabled: boolean;
  warmMinAge: string;
  warmReadOnly: boolean;
  warmForceMerge: boolean;
  warmMaxNumSegments: string;
  warmShrink: boolean;
  warmNumberOfShards: string;
  coldEnabled: boolean;
  coldMinAge: string;
  coldReadOnly: boolean;
  frozenEnabled: boolean;
  frozenMinAge: string;
  deleteEnabled: boolean;
  deleteMinAge: string;
};

const defaultPolicy = formatJson({
  policy: {
    phases: {
      hot: {
        actions: {
          rollover: {
            max_age: '30d',
            max_primary_shard_size: '50gb',
          },
        },
      },
      delete: {
        min_age: '90d',
        actions: {
          delete: {},
        },
      },
    },
  },
});

const defaultWizard: ILMWizardValues = {
  coldEnabled: false,
  coldMinAge: '60d',
  coldReadOnly: true,
  deleteEnabled: true,
  deleteMinAge: '90d',
  frozenEnabled: false,
  frozenMinAge: '120d',
  hotMaxAge: '30d',
  hotMaxDocs: '',
  hotMaxPrimaryShardSize: '50gb',
  hotMaxSize: '',
  hotRollover: true,
  warmEnabled: false,
  warmForceMerge: false,
  warmMaxNumSegments: '1',
  warmMinAge: '7d',
  warmNumberOfShards: '1',
  warmReadOnly: true,
  warmShrink: false,
};

type ILMFormValues = {
  body: string;
  name: string;
  wizard: ILMWizardValues;
};

export function ILMPoliciesPage({
  connection,
  initialPolicy,
  notify,
  refreshTick,
}: {
  connection: HostBodyWritable;
  initialPolicy?: string;
  notify: Notify;
  refreshTick: number;
}) {
  const [policies, setPolicies] = useState<IlmPolicy[]>([]);
  const [filter, setFilter] = useState('');
  const [deletePolicy, setDeletePolicy] = useState<IlmPolicy | null>(null);
  const [sort, setSort] = useState<SortState<ILMSortKey>>({ key: 'name', order: 'asc' });
  const [editorMode, setEditorMode] = useState<PolicyEditorMode>('wizard');
  const openedPolicy = useRef('');
  const form = useForm({
    defaultValues: { body: defaultPolicy, name: '', wizard: defaultWizard } satisfies ILMFormValues,
    onSubmit: async ({ value }) => {
      await save(value);
    },
  });

  useEffect(() => {
    void load();
  }, [connection, initialPolicy, refreshTick]);

  async function load() {
    try {
      const result = await ilmPoliciesList<true>({ path: clusterPath(connection), throwOnError: true });
      const items = (result.data.items ?? []).sort((left, right) => left.name.localeCompare(right.name));
      setPolicies(items);
      if (initialPolicy && openedPolicy.current !== initialPolicy) {
        const policy = items.find((item) => item.name === initialPolicy);
        if (policy) {
          edit(policy);
          openedPolicy.current = initialPolicy;
        }
      }
    } catch (error) {
      notify('danger', `Error loading ILM policies: ${errorMessage(error)}`);
    }
  }

  async function save(values: ILMFormValues) {
    const name = values.name.trim();
    const body = parseJson(values.body);
    if (!name) {
      notify('danger', 'ILM policy name is required');
      return;
    }
    if (!body || typeof body !== 'object' || Array.isArray(body)) {
      notify('danger', 'ILM policy body must be a JSON object');
      return;
    }
    try {
      await ilmPoliciesSave<true>({ body: { policy: body }, path: { ...clusterPath(connection), name }, throwOnError: true });
      notify('info', policies.some((policy) => policy.name === name) ? 'ILM policy successfully updated' : 'ILM policy successfully created');
      await load();
    } catch (error) {
      notify('danger', `Error saving ILM policy: ${errorMessage(error)}`);
    }
  }

  async function remove(policy: IlmPolicy) {
    try {
      await ilmPoliciesDelete<true>({ path: { ...clusterPath(connection), name: policy.name }, throwOnError: true });
      notify('info', 'ILM policy successfully deleted');
      await load();
    } catch (error) {
      notify('danger', `Error deleting ILM policy: ${errorMessage(error)}`);
    }
  }

  function edit(policy: IlmPolicy) {
    form.setFieldValue('name', policy.name);
    const body = formatJson({ policy: policy.policy ?? {} });
    form.setFieldValue('body', body);
    form.setFieldValue('wizard', wizardFromPolicyBody(body));
  }

  function resetForm() {
    form.setFieldValue('name', '');
    form.setFieldValue('body', defaultPolicy);
    form.setFieldValue('wizard', defaultWizard);
    setEditorMode('wizard');
  }

  function updateWizard(current: ILMWizardValues, next: Partial<ILMWizardValues>) {
    const value = { ...current, ...next };
    form.setFieldValue('wizard', value);
    form.setFieldValue('body', formatJson(policyBodyFromWizard(value)));
  }

  function switchMode(mode: PolicyEditorMode, body: string) {
    if (mode === 'wizard') {
      form.setFieldValue('wizard', wizardFromPolicyBody(body));
    }
    setEditorMode(mode);
  }

  const filtered = useMemo(
    () =>
      sortByText(
        policies.filter((policy) => policy.name.toLowerCase().includes(filter.toLowerCase()) || phasesText(policy).toLowerCase().includes(filter.toLowerCase())),
        sort,
        policySortValue,
      ),
    [filter, policies, sort],
  );
  const managedPolicies = filtered.filter(isManagedPolicy);
  const notManagedPolicies = filtered.filter((policy) => !isManagedPolicy(policy));

  return (
    <>
      {deletePolicy ? (
        <ConfirmModal
          body={
            <>
              Delete ILM policy <strong>{deletePolicy.name}</strong>? Existing indices can keep their lifecycle setting, but Elasticsearch will no longer find this policy.
            </>
          }
          confirmLabel={
            <>
              <Icon name="trash" /> delete policy
            </>
          }
          onClose={() => setDeletePolicy(null)}
          onConfirm={() => remove(deletePolicy)}
          title="delete ILM policy"
        />
      ) : null}
      <SplitPane
        storageKey="cerebro.ilmPoliciesSplitPercent"
        left={
          <>
            <div className="flex items-center justify-between gap-[15px]">
              <h4>
                ilm policies <small className="info-text">({filtered.length})</small>
              </h4>
              <Button icon="plus" size="xs" variant="success" onClick={resetForm}>
                new policy
              </Button>
            </div>
            <div className="form-group">
              <input className="form-control" placeholder="filter policies by name or phase" value={filter} onChange={(event) => setFilter(event.target.value)} />
            </div>
            <h4>my policies <small className="info-text">({notManagedPolicies.length})</small></h4>
            <ILMPolicyTable policies={notManagedPolicies} sort={sort} onDelete={setDeletePolicy} onSort={setSort} />

            <h4 className="mt-[20px]">managed policies <small className="info-text">({managedPolicies.length})</small></h4>
            <ILMPolicyTable policies={managedPolicies} sort={sort} onDelete={setDeletePolicy} onSort={setSort} />
          </>
        }
        right={
          <>
            <form.Subscribe selector={(state) => state.values.name}>
              {(name) => <h4>{policies.some((policy) => policy.name === name) ? `update policy ${name}` : 'create ilm policy'}</h4>}
            </form.Subscribe>
            <div className="row">
              <div className="col-xs-12">
                <div className="form-group">
                  <form.Field name="name">
                    {(field) => <input className="form-control" placeholder="policy name" value={field.state.value} onChange={(event) => field.handleChange(event.target.value)} />}
                  </form.Field>
                </div>
              </div>
              <div className="col-xs-12">
                <div className="form-group">
                  <form.Field name="body">
                    {(field) => (
                      <>
                        <div className="mb-[15px] inline-flex overflow-hidden border border-[#55595c]">
                          <button
                            className={`px-[12px] py-[6px] ${editorMode === 'wizard' ? 'bg-[#434749] text-white' : 'bg-transparent text-[#eceeef]'}`}
                            type="button"
                            onClick={() => switchMode('wizard', field.state.value)}
                          >
                            wizard
                          </button>
                          <button
                            className={`border-l border-[#55595c] px-[12px] py-[6px] ${editorMode === 'json' ? 'bg-[#434749] text-white' : 'bg-transparent text-[#eceeef]'}`}
                            type="button"
                            onClick={() => switchMode('json', field.state.value)}
                          >
                            json
                          </button>
                        </div>
                        {editorMode === 'wizard' ? (
                          <form.Subscribe selector={(state) => state.values.wizard}>
                            {(wizard) => <ILMPolicyWizard value={wizard} onChange={(next) => updateWizard(wizard, next)} />}
                          </form.Subscribe>
                        ) : (
                          <LazyJsonEditor height={600} value={field.state.value} onChange={field.handleChange} />
                        )}
                      </>
                    )}
                  </form.Field>
                </div>
              </div>
              <div className="col-xs-12 text-right">
                <form.Subscribe selector={(state) => state.values.name}>
                  {(name) => {
                    const editMode = policies.some((policy) => policy.name === name);
                    return (
                      <Button icon={editMode ? 'save' : 'plus'} type="submit" variant={editMode ? 'warning' : 'success'} onClick={() => void form.handleSubmit()}>
                        {editMode ? 'update' : 'create'}
                      </Button>
                    );
                  }}
                </form.Subscribe>
              </div>
            </div>
          </>
        }
      />
    </>
  );
}

function ILMPolicyTable({
  onDelete,
  onSort,
  policies,
  sort,
}: {
  onDelete: (policy: IlmPolicy) => void;
  onSort: (value: SortState<ILMSortKey> | ((value: SortState<ILMSortKey>) => SortState<ILMSortKey>)) => void;
  policies: IlmPolicy[];
  sort: SortState<ILMSortKey>;
}) {
  const columns: DataTableColumn<IlmPolicy>[] = [
    {
      className: 'break-all',
      header: sortButton('name', 'name', sort, onSort),
      key: 'name',
      render: (policy) => <><Icon name="history" /> {policy.name}</>,
    },
    {
      header: sortButton('phases', 'phases', sort, onSort),
      key: 'phases',
      render: (policy) => phasesText(policy) || <span className="info-text">none</span>,
    },
    {
      header: sortButton('version', 'version', sort, onSort),
      key: 'version',
      render: (policy) => textValue(policy.version) || 'n/a',
    },
    {
      header: 'used by',
      key: 'used-by',
      truncate: false,
      render: (policy) => <PolicyUsage policy={policy} />,
    },
    {
      className: 'text-right',
      header: 'actions',
      headerClassName: 'text-right',
      key: 'actions',
      render: (policy) => {
        const managed = isManagedPolicy(policy);
        return (
          <span className="inline-flex items-center gap-[10px]">
            <Link className="btn btn-default btn-xs" search={(previous) => ({ ...previous, policy: policy.name })} title="edit policy" to="/ilm">
              <Icon name="pencil" />
            </Link>
            <button
              className="btn btn-danger btn-xs"
              disabled={managed}
              title={managed ? 'policy is in use' : 'delete policy'}
              type="button"
              onClick={() => onDelete(policy)}
            >
              <Icon name="trash" />
            </button>
          </span>
        );
      },
    },
  ];

  return <DataTable columns={columns} getRowKey={(policy) => policy.name} rows={policies} />;
}

function ILMPolicyWizard({
  onChange,
  value,
}: {
  onChange: (next: Partial<ILMWizardValues>) => void;
  value: ILMWizardValues;
}) {
  return (
    <div className="space-y-[15px]">
      <WizardPhase title="hot phase" enabled locked>
        <WizardCheckbox checked={value.hotRollover} label="rollover" onChange={(hotRollover) => onChange({ hotRollover })} />
        {value.hotRollover ? (
          <div className="row">
            <WizardInput label="max age" placeholder="30d" value={value.hotMaxAge} onChange={(hotMaxAge) => onChange({ hotMaxAge })} />
            <WizardInput label="max primary shard size" placeholder="50gb" value={value.hotMaxPrimaryShardSize} onChange={(hotMaxPrimaryShardSize) => onChange({ hotMaxPrimaryShardSize })} />
            <WizardInput label="max size" placeholder="100gb" value={value.hotMaxSize} onChange={(hotMaxSize) => onChange({ hotMaxSize })} />
            <WizardInput label="max docs" placeholder="1000000" value={value.hotMaxDocs} onChange={(hotMaxDocs) => onChange({ hotMaxDocs })} />
          </div>
        ) : null}
      </WizardPhase>

      <WizardPhase title="warm phase" enabled={value.warmEnabled} onToggle={(warmEnabled) => onChange({ warmEnabled })}>
        <div className="row">
          <WizardInput label="min age" placeholder="7d" value={value.warmMinAge} onChange={(warmMinAge) => onChange({ warmMinAge })} />
        </div>
        <WizardCheckbox checked={value.warmReadOnly} label="read only" onChange={(warmReadOnly) => onChange({ warmReadOnly })} />
        <WizardCheckbox checked={value.warmForceMerge} label="force merge" onChange={(warmForceMerge) => onChange({ warmForceMerge })} />
        {value.warmForceMerge ? (
          <div className="row">
            <WizardInput label="max segments" placeholder="1" value={value.warmMaxNumSegments} onChange={(warmMaxNumSegments) => onChange({ warmMaxNumSegments })} />
          </div>
        ) : null}
        <WizardCheckbox checked={value.warmShrink} label="shrink" onChange={(warmShrink) => onChange({ warmShrink })} />
        {value.warmShrink ? (
          <div className="row">
            <WizardInput label="number of shards" placeholder="1" value={value.warmNumberOfShards} onChange={(warmNumberOfShards) => onChange({ warmNumberOfShards })} />
          </div>
        ) : null}
      </WizardPhase>

      <WizardPhase title="cold phase" enabled={value.coldEnabled} onToggle={(coldEnabled) => onChange({ coldEnabled })}>
        <div className="row">
          <WizardInput label="min age" placeholder="60d" value={value.coldMinAge} onChange={(coldMinAge) => onChange({ coldMinAge })} />
        </div>
        <WizardCheckbox checked={value.coldReadOnly} label="read only" onChange={(coldReadOnly) => onChange({ coldReadOnly })} />
      </WizardPhase>

      <WizardPhase title="frozen phase" enabled={value.frozenEnabled} onToggle={(frozenEnabled) => onChange({ frozenEnabled })}>
        <div className="row">
          <WizardInput label="min age" placeholder="120d" value={value.frozenMinAge} onChange={(frozenMinAge) => onChange({ frozenMinAge })} />
        </div>
      </WizardPhase>

      <WizardPhase title="delete phase" enabled={value.deleteEnabled} onToggle={(deleteEnabled) => onChange({ deleteEnabled })}>
        <div className="row">
          <WizardInput label="min age" placeholder="90d" value={value.deleteMinAge} onChange={(deleteMinAge) => onChange({ deleteMinAge })} />
        </div>
      </WizardPhase>
    </div>
  );
}

function WizardPhase({
  children,
  enabled,
  locked = false,
  onToggle,
  title,
}: {
  children: ReactNode;
  enabled: boolean;
  locked?: boolean;
  onToggle?: (enabled: boolean) => void;
  title: string;
}) {
  return (
    <section className="border border-[#55595c] p-[12px]">
      <div className="mb-[10px] flex items-center justify-between">
        <h4 className="!m-0">{title}</h4>
        {locked ? (
          <span className="label label-success">enabled</span>
        ) : (
          <Checkbox checked={enabled} label="enabled" onChange={(checked) => onToggle?.(checked)} />
        )}
      </div>
      {enabled ? children : <div className="info-text">disabled</div>}
    </section>
  );
}

function WizardInput({
  label,
  onChange,
  placeholder,
  value,
}: {
  label: string;
  onChange: (value: string) => void;
  placeholder: string;
  value: string;
}) {
  return (
    <div className="col-sm-6 form-group">
      <label className="form-label">{label}</label>
      <input className="form-control font-mono" placeholder={placeholder} value={value} onChange={(event) => onChange(event.target.value)} />
    </div>
  );
}

function WizardCheckbox({
  checked,
  label,
  onChange,
}: {
  checked: boolean;
  label: string;
  onChange: (value: boolean) => void;
}) {
  return (
    <div className="checkbox">
      <Checkbox checked={checked} label={label} onChange={onChange} />
    </div>
  );
}

function policyBodyFromWizard(value: ILMWizardValues) {
  const phases: Record<string, unknown> = {
    hot: { actions: hotActions(value) },
  };
  if (value.warmEnabled) {
    phases.warm = phaseBody(value.warmMinAge, warmActions(value));
  }
  if (value.coldEnabled) {
    phases.cold = phaseBody(value.coldMinAge, value.coldReadOnly ? { readonly: {} } : {});
  }
  if (value.frozenEnabled) {
    phases.frozen = phaseBody(value.frozenMinAge, {});
  }
  if (value.deleteEnabled) {
    phases.delete = phaseBody(value.deleteMinAge, { delete: {} });
  }
  return { policy: { phases } };
}

function hotActions(value: ILMWizardValues) {
  if (!value.hotRollover) return {};
  const rollover: Record<string, unknown> = {};
  setString(rollover, 'max_age', value.hotMaxAge);
  setString(rollover, 'max_primary_shard_size', value.hotMaxPrimaryShardSize);
  setString(rollover, 'max_size', value.hotMaxSize);
  setNumber(rollover, 'max_docs', value.hotMaxDocs);
  return Object.keys(rollover).length ? { rollover } : {};
}

function warmActions(value: ILMWizardValues) {
  const actions: Record<string, unknown> = {};
  if (value.warmReadOnly) {
    actions.readonly = {};
  }
  if (value.warmForceMerge) {
    const forcemerge: Record<string, unknown> = {};
    setNumber(forcemerge, 'max_num_segments', value.warmMaxNumSegments);
    actions.forcemerge = forcemerge;
  }
  if (value.warmShrink) {
    const shrink: Record<string, unknown> = {};
    setNumber(shrink, 'number_of_shards', value.warmNumberOfShards);
    actions.shrink = shrink;
  }
  return actions;
}

function phaseBody(minAge: string, actions: Record<string, unknown>) {
  const phase: Record<string, unknown> = { actions };
  setString(phase, 'min_age', minAge);
  return phase;
}

function wizardFromPolicyBody(body: string): ILMWizardValues {
  const parsed = parseJson(body);
  const root = objectValue(parsed);
  const policy = objectValue(root.policy);
  const phases = objectValue(policy.phases);
  const hot = objectValue(phases.hot);
  const hotActionsBody = objectValue(hot.actions);
  const rollover = objectValue(hotActionsBody.rollover);
  const warm = objectValue(phases.warm);
  const warmActionsBody = objectValue(warm.actions);
  const forcemerge = objectValue(warmActionsBody.forcemerge);
  const shrink = objectValue(warmActionsBody.shrink);
  const cold = objectValue(phases.cold);
  const coldActionsBody = objectValue(cold.actions);
  const frozen = objectValue(phases.frozen);
  const deletePhase = objectValue(phases.delete);
  return {
    coldEnabled: Boolean(phases.cold),
    coldMinAge: stringField(cold.min_age, defaultWizard.coldMinAge),
    coldReadOnly: Boolean(coldActionsBody.readonly),
    deleteEnabled: Boolean(phases.delete),
    deleteMinAge: stringField(deletePhase.min_age, defaultWizard.deleteMinAge),
    frozenEnabled: Boolean(phases.frozen),
    frozenMinAge: stringField(frozen.min_age, defaultWizard.frozenMinAge),
    hotMaxAge: stringField(rollover.max_age, defaultWizard.hotMaxAge),
    hotMaxDocs: stringField(rollover.max_docs, defaultWizard.hotMaxDocs),
    hotMaxPrimaryShardSize: stringField(rollover.max_primary_shard_size, defaultWizard.hotMaxPrimaryShardSize),
    hotMaxSize: stringField(rollover.max_size, defaultWizard.hotMaxSize),
    hotRollover: Boolean(hotActionsBody.rollover),
    warmEnabled: Boolean(phases.warm),
    warmForceMerge: Boolean(warmActionsBody.forcemerge),
    warmMaxNumSegments: stringField(forcemerge.max_num_segments, defaultWizard.warmMaxNumSegments),
    warmMinAge: stringField(warm.min_age, defaultWizard.warmMinAge),
    warmNumberOfShards: stringField(shrink.number_of_shards, defaultWizard.warmNumberOfShards),
    warmReadOnly: Boolean(warmActionsBody.readonly),
    warmShrink: Boolean(warmActionsBody.shrink),
  };
}

function objectValue(value: unknown): Record<string, unknown> {
  return value && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : {};
}

function stringField(value: unknown, fallback: string) {
  const text = textValue(value);
  return text || fallback;
}

function setString(target: Record<string, unknown>, key: string, value: string) {
  const trimmed = value.trim();
  if (trimmed) {
    target[key] = trimmed;
  }
}

function setNumber(target: Record<string, unknown>, key: string, value: string) {
  const trimmed = value.trim();
  if (!trimmed) return;
  const parsed = Number(trimmed);
  target[key] = Number.isFinite(parsed) ? parsed : trimmed;
}

function sortButton(
  key: ILMSortKey,
  label: string,
  sort: SortState<ILMSortKey>,
  setSort: (value: SortState<ILMSortKey> | ((value: SortState<ILMSortKey>) => SortState<ILMSortKey>)) => void,
) {
  return (
    <button className="normal-action border-0 bg-transparent p-0 text-inherit" type="button" onClick={() => setSort((value) => nextSort(value, key))}>
      {label} <SortIndicator active={sort.key === key} order={sort.order} />
    </button>
  );
}

function phasesText(policy: IlmPolicy) {
  return (policy.phases ?? []).join(', ');
}

function isManagedPolicy(policy: IlmPolicy) {
  const usedBy = policy.in_use_by;
  return Boolean(
    usedBy?.indices?.length
      || usedBy?.data_streams?.length
      || usedBy?.composable_templates?.length,
  );
}

function managedPolicyUsage(policy: IlmPolicy) {
  const usedBy = policy.in_use_by;
  const parts = [
    usedBy?.indices?.length ? `${usedBy.indices.length} indices` : '',
    usedBy?.data_streams?.length ? `${usedBy.data_streams.length} data streams` : '',
    usedBy?.composable_templates?.length ? `${usedBy.composable_templates.length} templates` : '',
  ].filter(Boolean);
  return parts.join(', ');
}

function PolicyUsage({ policy }: { policy: IlmPolicy }) {
  if (!isManagedPolicy(policy)) return <span className="info-text">none</span>;
  const usedBy = policy.in_use_by;
  return (
    <span className="group relative inline-block">
      <button className="normal-action border-0 bg-transparent p-0 text-left text-inherit" type="button">
        {managedPolicyUsage(policy)} <Icon name="caret-down" />
      </button>
      <span className="absolute left-0 top-full z-[1000] hidden w-[360px] border border-[#55595c] bg-[#373a3c] p-[10px] text-left shadow-lg group-hover:block group-focus-within:block">
        <UsageGroup label="indices" values={usedBy?.indices ?? []} />
        <UsageGroup label="data streams" values={usedBy?.data_streams ?? []} />
        <UsageGroup label="templates" values={usedBy?.composable_templates ?? []} />
      </span>
    </span>
  );
}

function UsageGroup({ label, values }: { label: string; values: string[] }) {
  if (!values.length) return null;
  const visible = values.slice(0, 12);
  const hidden = values.length - visible.length;
  return (
    <span className="mb-[8px] block last:mb-0">
      <span className="mb-[3px] block text-[11px] uppercase text-[#8b8f95]">
        {label} ({values.length})
      </span>
      <span className="block max-h-[140px] overflow-auto pr-[4px]">
        {visible.map((value) => (
          <span className="block break-all font-mono text-[12px] text-[#eceeef]" key={value}>
            {value}
          </span>
        ))}
        {hidden > 0 ? <span className="block info-text">+ {hidden} more</span> : null}
      </span>
    </span>
  );
}

function policySortValue(policy: IlmPolicy, key: ILMSortKey) {
  switch (key) {
    case 'phases':
      return phasesText(policy);
    case 'version':
      return textValue(policy.version).padStart(12, '0');
    default:
      return policy.name;
  }
}
