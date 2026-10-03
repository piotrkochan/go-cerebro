import { useEffect, useMemo, useRef, useState } from 'react';
import { Link } from '@tanstack/react-router';
import { useForm } from '@tanstack/react-form';

import { ilmPoliciesDelete, ilmPoliciesList, ilmPoliciesSave } from '../api/ilmClient';
import type { HostBodyWritable, IlmPolicy } from '../api/client/types.gen';
import { Button } from '../components/Button';
import { DataTable, SortIndicator, type DataTableColumn } from '../components/DataTable';
import { Icon } from '../components/Icon';
import { ILMPolicyWizard } from '../components/ILMPolicyWizard';
import { LazyJsonEditor } from '../components/LazyJsonEditor';
import { ConfirmModal } from '../components/Modal';
import { SplitPane } from '../components/SplitPane';
import {
  createDefaultILMWizard,
  ilmFeaturesForVersion,
  policyBodyFromWizard,
  validateILMWizard,
  wizardFromPolicyBody,
  type ILMWizardValues,
} from '../forms/ilmPolicyForm';
import type { Notify } from '../stores/alertsStore';
import { clusterPath } from '../utils/connection';
import { errorMessage, formatJson, parseJson, textValue } from '../utils/format';
import { nextSort, sortByText, type SortState } from '../utils/sort';

type ILMSortKey = 'name' | 'phases' | 'version';
type PolicyEditorMode = 'wizard' | 'json';

const defaultPolicy = formatJson(policyBodyFromWizard(createDefaultILMWizard(), '{}'));

type ILMFormValues = {
  body: string;
  name: string;
  wizard: ILMWizardValues;
};

export function ILMPoliciesPage({
  connection,
  elasticsearchVersion,
  initialPolicy,
  notify,
  refreshTick,
}: {
  connection: HostBodyWritable;
  elasticsearchVersion?: string;
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
  const wizardFeatures = useMemo(() => ilmFeaturesForVersion(elasticsearchVersion), [elasticsearchVersion]);
  const form = useForm({
    defaultValues: { body: defaultPolicy, name: '', wizard: createDefaultILMWizard() } satisfies ILMFormValues,
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
    if (editorMode === 'wizard') {
      const errors = validateILMWizard(values.wizard, wizardFeatures);
      if (errors.length) {
        notify('danger', errors[0]);
        return;
      }
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
    const wizard = createDefaultILMWizard();
    form.setFieldValue('name', '');
    form.setFieldValue('body', formatJson(policyBodyFromWizard(wizard, '{}')));
    form.setFieldValue('wizard', wizard);
    setEditorMode('wizard');
  }

  function updateWizard(value: ILMWizardValues, currentBody: string) {
    form.setFieldValue('wizard', value);
    form.setFieldValue('body', formatJson(policyBodyFromWizard(value, currentBody)));
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
                            {(wizard) => (
                              <ILMPolicyWizard
                                features={wizardFeatures}
                                value={wizard}
                                onChange={(next) => updateWizard(next, field.state.value)}
                              />
                            )}
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
