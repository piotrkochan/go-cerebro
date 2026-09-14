export type BooleanChoice = 'default' | 'true' | 'false';
export type MigrateChoice = 'automatic' | 'enabled' | 'disabled';
export type SamplingMethod = 'default' | 'aggregate' | 'last_value';

export type AllocationRule = {
  attribute: string;
  value: string;
};

export type ILMPhaseWizard = {
  allocate: {
    enabled: boolean;
    exclude: AllocationRule[];
    include: AllocationRule[];
    numberOfReplicas: string;
    require: AllocationRule[];
    totalShardsPerNode: string;
  };
  delete: {
    deleteSearchableSnapshot: BooleanChoice;
    enabled: boolean;
  };
  downsample: {
    enabled: boolean;
    fixedInterval: string;
    forceMergeIndex: BooleanChoice;
    samplingMethod: SamplingMethod;
  };
  enabled: boolean;
  forceMerge: {
    enabled: boolean;
    maxNumSegments: string;
  };
  migrate: MigrateChoice;
  minAge: string;
  readOnly: boolean;
  rollover: {
    enabled: boolean;
    maxAge: string;
    maxDocs: string;
    maxPrimaryShardDocs: string;
    maxPrimaryShardSize: string;
    maxSize: string;
    minAge: string;
    minDocs: string;
    minPrimaryShardDocs: string;
    minPrimaryShardSize: string;
    minSize: string;
  };
  searchableSnapshot: {
    enabled: boolean;
    forceMergeIndex: BooleanChoice;
    replicateFor: string;
    snapshotRepository: string;
    totalShardsPerNode: string;
  };
  setPriority: {
    enabled: boolean;
    priority: string;
  };
  shrink: {
    enabled: boolean;
    maxPrimaryShardSize: string;
    numberOfShards: string;
  };
  unfollow: boolean;
  waitForSnapshot: {
    enabled: boolean;
    policy: string;
  };
};

export type ILMWizardValues = {
  cold: ILMPhaseWizard;
  delete: ILMPhaseWizard;
  frozen: ILMPhaseWizard;
  hot: ILMPhaseWizard;
  warm: ILMPhaseWizard;
};

export type ILMWizardFeatures = {
  downsample: boolean;
  downsampleForceMerge: boolean;
  frozenPhase: boolean;
  migrate: boolean;
  searchableSnapshot: boolean;
  waitForSnapshot: boolean;
};

type PhaseName = keyof ILMWizardValues;
type RecordValue = Record<string, unknown>;

const phaseNames: PhaseName[] = ['hot', 'warm', 'cold', 'frozen', 'delete'];
const managedActions = [
  'allocate',
  'delete',
  'downsample',
  'forcemerge',
  'migrate',
  'readonly',
  'rollover',
  'searchable_snapshot',
  'set_priority',
  'shrink',
  'unfollow',
  'wait_for_snapshot',
] as const;

export function createDefaultILMWizard(): ILMWizardValues {
  return {
    cold: createPhase({ enabled: false, minAge: '60d', priority: '0', readOnly: true }),
    delete: createPhase({ deleteAction: true, enabled: true, minAge: '90d' }),
    frozen: createPhase({ enabled: false, minAge: '120d' }),
    hot: createPhase({ enabled: true, priority: '100', rollover: true }),
    warm: createPhase({ enabled: false, minAge: '7d', priority: '50', readOnly: true }),
  };
}

export function ilmFeaturesForVersion(version: string | undefined): ILMWizardFeatures {
  return {
    downsample: versionAtLeast(version, 8, 7),
    downsampleForceMerge: versionAtLeast(version, 9, 3),
    frozenPhase: versionAtLeast(version, 7, 10),
    migrate: versionAtLeast(version, 7, 10),
    searchableSnapshot: versionAtLeast(version, 7, 10),
    waitForSnapshot: versionAtLeast(version, 7, 4),
  };
}

export function policyBodyFromWizard(value: ILMWizardValues, currentBody: string): RecordValue {
  const root = cloneRecord(parseRecord(currentBody));
  const policy = cloneRecord(objectValue(root.policy));
  const phases = cloneRecord(objectValue(policy.phases));

  for (const phaseName of phaseNames) {
    const wizardPhase = value[phaseName];
    if (!wizardPhase.enabled) {
      delete phases[phaseName];
      continue;
    }

    const phase = cloneRecord(objectValue(phases[phaseName]));
    const actions = cloneRecord(objectValue(phase.actions));
    updateManagedActions(actions, wizardPhase);
    phase.actions = actions;
    if (phaseName === 'hot') {
      delete phase.min_age;
    } else {
      setOptionalString(phase, 'min_age', wizardPhase.minAge);
    }
    phases[phaseName] = phase;
  }

  policy.phases = phases;
  root.policy = policy;
  return root;
}

export function wizardFromPolicyBody(body: string): ILMWizardValues {
  const defaults = createDefaultILMWizard();
  const root = parseRecord(body);
  const phases = objectValue(objectValue(root.policy).phases);

  return {
    cold: phaseFromPolicy(phases, 'cold', defaults.cold),
    delete: phaseFromPolicy(phases, 'delete', defaults.delete),
    frozen: phaseFromPolicy(phases, 'frozen', defaults.frozen),
    hot: phaseFromPolicy(phases, 'hot', defaults.hot),
    warm: phaseFromPolicy(phases, 'warm', defaults.warm),
  };
}

export function validateILMWizard(value: ILMWizardValues, features: ILMWizardFeatures): string[] {
  const errors: string[] = [];
  const phases: Array<[PhaseName, ILMPhaseWizard]> = phaseNames.map((name) => [name, value[name]]);

  for (const [name, phase] of phases) {
    if (!phase.enabled) continue;
    if (phase.forceMerge.enabled) validateRequiredPositiveInteger(errors, `${name} force merge max segments`, phase.forceMerge.maxNumSegments);
    validatePriority(errors, name, phase);
    validateShrink(errors, name, phase);
    validateAllocate(errors, name, phase);
    validateRolloverNumbers(errors, name, phase);
    if (features.downsample && phase.downsample.enabled && !phase.downsample.fixedInterval.trim()) {
      errors.push(`${name} downsample requires a fixed interval`);
    }
    if (features.searchableSnapshot && phase.searchableSnapshot.enabled && !phase.searchableSnapshot.snapshotRepository.trim()) {
      errors.push(`${name} searchable snapshot requires a repository`);
    }
    if (features.searchableSnapshot && phase.searchableSnapshot.enabled && phase.searchableSnapshot.totalShardsPerNode.trim()) {
      validatePositiveInteger(errors, `${name} searchable snapshot shards per node`, phase.searchableSnapshot.totalShardsPerNode);
    }
  }

  if (value.hot.enabled && value.hot.rollover.enabled && !hasRolloverMaximum(value.hot.rollover)) {
    errors.push('hot rollover requires at least one maximum condition');
  }
  if (value.hot.enabled && !value.hot.rollover.enabled
    && (value.hot.readOnly || value.hot.downsample.enabled || value.hot.shrink.enabled || value.hot.forceMerge.enabled)) {
    errors.push('hot read only, downsample, shrink and force merge actions require rollover');
  }
  if (features.searchableSnapshot
    && value.hot.enabled && value.hot.searchableSnapshot.enabled
    && value.cold.enabled && value.cold.searchableSnapshot.enabled) {
    errors.push('searchable snapshot cannot be enabled in both hot and cold phases');
  }
  if (features.searchableSnapshot && value.hot.enabled && value.hot.searchableSnapshot.enabled
    && value.warm.enabled && (value.warm.shrink.enabled || value.warm.forceMerge.enabled)) {
    errors.push('warm shrink and force merge cannot follow a hot searchable snapshot');
  }
  if (features.waitForSnapshot && value.delete.enabled && value.delete.waitForSnapshot.enabled && !value.delete.waitForSnapshot.policy.trim()) {
    errors.push('delete wait for snapshot requires an SLM policy');
  }
  if (features.downsampleForceMerge) {
    const samplingMethods = new Set(
      phases
        .filter(([, phase]) => phase.enabled && phase.downsample.enabled && phase.downsample.samplingMethod !== 'default')
        .map(([, phase]) => phase.downsample.samplingMethod),
    );
    if (samplingMethods.size > 1) errors.push('all downsample actions must use the same sampling method');
  }
  return errors;
}

function createPhase({
  deleteAction = false,
  enabled,
  minAge = '',
  priority = '',
  readOnly = false,
  rollover = false,
}: {
  deleteAction?: boolean;
  enabled: boolean;
  minAge?: string;
  priority?: string;
  readOnly?: boolean;
  rollover?: boolean;
}): ILMPhaseWizard {
  return {
    allocate: { enabled: false, exclude: [], include: [], numberOfReplicas: '', require: [], totalShardsPerNode: '' },
    delete: { deleteSearchableSnapshot: 'default', enabled: deleteAction },
    downsample: { enabled: false, fixedInterval: '1h', forceMergeIndex: 'default', samplingMethod: 'default' },
    enabled,
    forceMerge: { enabled: false, maxNumSegments: '1' },
    migrate: 'automatic',
    minAge,
    readOnly,
    rollover: {
      enabled: rollover,
      maxAge: '30d',
      maxDocs: '',
      maxPrimaryShardDocs: '',
      maxPrimaryShardSize: '50gb',
      maxSize: '',
      minAge: '',
      minDocs: '',
      minPrimaryShardDocs: '',
      minPrimaryShardSize: '',
      minSize: '',
    },
    searchableSnapshot: {
      enabled: false,
      forceMergeIndex: 'default',
      replicateFor: '',
      snapshotRepository: '',
      totalShardsPerNode: '',
    },
    setPriority: { enabled: false, priority },
    shrink: { enabled: false, maxPrimaryShardSize: '', numberOfShards: '1' },
    unfollow: false,
    waitForSnapshot: { enabled: false, policy: '' },
  };
}

function phaseFromPolicy(phases: RecordValue, name: PhaseName, defaults: ILMPhaseWizard): ILMPhaseWizard {
  const exists = hasOwn(phases, name);
  const phase = objectValue(phases[name]);
  const actions = objectValue(phase.actions);
  const rollover = objectValue(actions.rollover);
  const allocate = objectValue(actions.allocate);
  const downsample = objectValue(actions.downsample);
  const forceMerge = objectValue(actions.forcemerge);
  const searchableSnapshot = objectValue(actions.searchable_snapshot);
  const setPriority = objectValue(actions.set_priority);
  const shrink = objectValue(actions.shrink);
  const waitForSnapshot = objectValue(actions.wait_for_snapshot);
  const deleteAction = objectValue(actions.delete);

  return {
    allocate: {
      enabled: hasOwn(actions, 'allocate'),
      exclude: allocationRules(allocate.exclude),
      include: allocationRules(allocate.include),
      numberOfReplicas: actionField(actions, 'allocate', allocate, 'number_of_replicas', defaults.allocate.numberOfReplicas),
      require: allocationRules(allocate.require),
      totalShardsPerNode: actionField(actions, 'allocate', allocate, 'total_shards_per_node', defaults.allocate.totalShardsPerNode),
    },
    delete: {
      deleteSearchableSnapshot: booleanChoice(deleteAction, 'delete_searchable_snapshot'),
      enabled: hasOwn(actions, 'delete'),
    },
    downsample: {
      enabled: hasOwn(actions, 'downsample'),
      fixedInterval: actionField(actions, 'downsample', downsample, 'fixed_interval', defaults.downsample.fixedInterval),
      forceMergeIndex: booleanChoice(downsample, 'force_merge_index'),
      samplingMethod: samplingMethod(downsample.sampling_method),
    },
    enabled: exists,
    forceMerge: {
      enabled: hasOwn(actions, 'forcemerge'),
      maxNumSegments: actionField(actions, 'forcemerge', forceMerge, 'max_num_segments', defaults.forceMerge.maxNumSegments),
    },
    migrate: migrateChoice(actions),
    minAge: exists ? text(phase.min_age) : defaults.minAge,
    readOnly: hasOwn(actions, 'readonly'),
    rollover: {
      enabled: hasOwn(actions, 'rollover'),
      maxAge: actionField(actions, 'rollover', rollover, 'max_age', defaults.rollover.maxAge),
      maxDocs: actionField(actions, 'rollover', rollover, 'max_docs', defaults.rollover.maxDocs),
      maxPrimaryShardDocs: actionField(actions, 'rollover', rollover, 'max_primary_shard_docs', defaults.rollover.maxPrimaryShardDocs),
      maxPrimaryShardSize: actionField(actions, 'rollover', rollover, 'max_primary_shard_size', defaults.rollover.maxPrimaryShardSize),
      maxSize: actionField(actions, 'rollover', rollover, 'max_size', defaults.rollover.maxSize),
      minAge: actionField(actions, 'rollover', rollover, 'min_age', defaults.rollover.minAge),
      minDocs: actionField(actions, 'rollover', rollover, 'min_docs', defaults.rollover.minDocs),
      minPrimaryShardDocs: actionField(actions, 'rollover', rollover, 'min_primary_shard_docs', defaults.rollover.minPrimaryShardDocs),
      minPrimaryShardSize: actionField(actions, 'rollover', rollover, 'min_primary_shard_size', defaults.rollover.minPrimaryShardSize),
      minSize: actionField(actions, 'rollover', rollover, 'min_size', defaults.rollover.minSize),
    },
    searchableSnapshot: {
      enabled: hasOwn(actions, 'searchable_snapshot'),
      forceMergeIndex: booleanChoice(searchableSnapshot, 'force_merge_index'),
      replicateFor: actionField(actions, 'searchable_snapshot', searchableSnapshot, 'replicate_for', defaults.searchableSnapshot.replicateFor),
      snapshotRepository: actionField(actions, 'searchable_snapshot', searchableSnapshot, 'snapshot_repository', defaults.searchableSnapshot.snapshotRepository),
      totalShardsPerNode: actionField(actions, 'searchable_snapshot', searchableSnapshot, 'total_shards_per_node', defaults.searchableSnapshot.totalShardsPerNode),
    },
    setPriority: {
      enabled: hasOwn(actions, 'set_priority'),
      priority: actionField(actions, 'set_priority', setPriority, 'priority', defaults.setPriority.priority),
    },
    shrink: {
      enabled: hasOwn(actions, 'shrink'),
      maxPrimaryShardSize: actionField(actions, 'shrink', shrink, 'max_primary_shard_size', defaults.shrink.maxPrimaryShardSize),
      numberOfShards: actionField(actions, 'shrink', shrink, 'number_of_shards', defaults.shrink.numberOfShards),
    },
    unfollow: hasOwn(actions, 'unfollow'),
    waitForSnapshot: {
      enabled: hasOwn(actions, 'wait_for_snapshot'),
      policy: actionField(actions, 'wait_for_snapshot', waitForSnapshot, 'policy', defaults.waitForSnapshot.policy),
    },
  };
}

function updateManagedActions(actions: RecordValue, phase: ILMPhaseWizard) {
  updateAction(actions, 'allocate', phase.allocate.enabled, ['number_of_replicas', 'total_shards_per_node', 'include', 'exclude', 'require'], (action) => {
    setOptionalNumber(action, 'number_of_replicas', phase.allocate.numberOfReplicas);
    setOptionalNumber(action, 'total_shards_per_node', phase.allocate.totalShardsPerNode);
    setAllocationRules(action, 'include', phase.allocate.include);
    setAllocationRules(action, 'exclude', phase.allocate.exclude);
    setAllocationRules(action, 'require', phase.allocate.require);
  });
  updateAction(actions, 'delete', phase.delete.enabled, ['delete_searchable_snapshot'], (action) => {
    setBooleanChoice(action, 'delete_searchable_snapshot', phase.delete.deleteSearchableSnapshot);
  });
  updateAction(actions, 'downsample', phase.downsample.enabled, ['fixed_interval', 'force_merge_index', 'sampling_method'], (action) => {
    setOptionalString(action, 'fixed_interval', phase.downsample.fixedInterval);
    setBooleanChoice(action, 'force_merge_index', phase.downsample.forceMergeIndex);
    if (phase.downsample.samplingMethod !== 'default') action.sampling_method = phase.downsample.samplingMethod;
  });
  updateAction(actions, 'forcemerge', phase.forceMerge.enabled, ['max_num_segments'], (action) => {
    setOptionalNumber(action, 'max_num_segments', phase.forceMerge.maxNumSegments);
  });
  updateMigrateAction(actions, phase.migrate);
  updateAction(actions, 'readonly', phase.readOnly, [], () => undefined);
  updateAction(actions, 'rollover', phase.rollover.enabled, rolloverKeys, (action) => {
    setOptionalString(action, 'max_age', phase.rollover.maxAge);
    setOptionalNumber(action, 'max_docs', phase.rollover.maxDocs);
    setOptionalNumber(action, 'max_primary_shard_docs', phase.rollover.maxPrimaryShardDocs);
    setOptionalString(action, 'max_primary_shard_size', phase.rollover.maxPrimaryShardSize);
    setOptionalString(action, 'max_size', phase.rollover.maxSize);
    setOptionalString(action, 'min_age', phase.rollover.minAge);
    setOptionalNumber(action, 'min_docs', phase.rollover.minDocs);
    setOptionalNumber(action, 'min_primary_shard_docs', phase.rollover.minPrimaryShardDocs);
    setOptionalString(action, 'min_primary_shard_size', phase.rollover.minPrimaryShardSize);
    setOptionalString(action, 'min_size', phase.rollover.minSize);
  });
  updateAction(actions, 'searchable_snapshot', phase.searchableSnapshot.enabled, ['snapshot_repository', 'replicate_for', 'force_merge_index', 'total_shards_per_node'], (action) => {
    setOptionalString(action, 'snapshot_repository', phase.searchableSnapshot.snapshotRepository);
    setOptionalString(action, 'replicate_for', phase.searchableSnapshot.replicateFor);
    setBooleanChoice(action, 'force_merge_index', phase.searchableSnapshot.forceMergeIndex);
    setOptionalNumber(action, 'total_shards_per_node', phase.searchableSnapshot.totalShardsPerNode);
  });
  updateAction(actions, 'set_priority', phase.setPriority.enabled, ['priority'], (action) => {
    setOptionalNullableNumber(action, 'priority', phase.setPriority.priority);
  });
  updateAction(actions, 'shrink', phase.shrink.enabled, ['number_of_shards', 'max_primary_shard_size'], (action) => {
    setOptionalNumber(action, 'number_of_shards', phase.shrink.numberOfShards);
    setOptionalString(action, 'max_primary_shard_size', phase.shrink.maxPrimaryShardSize);
  });
  updateAction(actions, 'unfollow', phase.unfollow, [], () => undefined);
  updateAction(actions, 'wait_for_snapshot', phase.waitForSnapshot.enabled, ['policy'], (action) => {
    setOptionalString(action, 'policy', phase.waitForSnapshot.policy);
  });
}

const rolloverKeys = [
  'max_age', 'max_docs', 'max_primary_shard_docs', 'max_primary_shard_size', 'max_size',
  'min_age', 'min_docs', 'min_primary_shard_docs', 'min_primary_shard_size', 'min_size',
];

function updateAction(actions: RecordValue, name: typeof managedActions[number], enabled: boolean, keys: string[], apply: (action: RecordValue) => void) {
  if (!enabled) {
    delete actions[name];
    return;
  }
  const action = cloneRecord(objectValue(actions[name]));
  for (const key of keys) delete action[key];
  apply(action);
  actions[name] = action;
}

function updateMigrateAction(actions: RecordValue, choice: MigrateChoice) {
  if (choice === 'automatic') {
    delete actions.migrate;
    return;
  }
  const action = cloneRecord(objectValue(actions.migrate));
  if (choice === 'disabled') action.enabled = false;
  else delete action.enabled;
  actions.migrate = action;
}

function actionField(actions: RecordValue, actionName: string, action: RecordValue, field: string, fallback: string) {
  return hasOwn(actions, actionName) ? text(action[field]) : fallback;
}

function allocationRules(value: unknown): AllocationRule[] {
  return Object.entries(objectValue(value))
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([attribute, ruleValue]) => ({ attribute, value: text(ruleValue) }));
}

function setAllocationRules(action: RecordValue, name: 'exclude' | 'include' | 'require', rules: AllocationRule[]) {
  const entries = rules
    .map((rule) => [rule.attribute.trim(), rule.value.trim()] as const)
    .filter(([attribute]) => Boolean(attribute));
  if (entries.length) action[name] = Object.fromEntries(entries);
}

function booleanChoice(value: RecordValue, key: string): BooleanChoice {
  if (!hasOwn(value, key)) return 'default';
  return value[key] === false ? 'false' : 'true';
}

function samplingMethod(value: unknown): SamplingMethod {
  return value === 'aggregate' || value === 'last_value' ? value : 'default';
}

function migrateChoice(actions: RecordValue): MigrateChoice {
  if (!hasOwn(actions, 'migrate')) return 'automatic';
  return objectValue(actions.migrate).enabled === false ? 'disabled' : 'enabled';
}

function setBooleanChoice(target: RecordValue, key: string, value: BooleanChoice) {
  if (value !== 'default') target[key] = value === 'true';
}

function setOptionalString(target: RecordValue, key: string, value: string) {
  const trimmed = value.trim();
  if (trimmed) target[key] = trimmed;
  else delete target[key];
}

function setOptionalNumber(target: RecordValue, key: string, value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    delete target[key];
    return;
  }
  const parsed = Number(trimmed);
  target[key] = Number.isFinite(parsed) ? parsed : trimmed;
}

function setOptionalNullableNumber(target: RecordValue, key: string, value: string) {
  if (value.trim() === 'null') {
    target[key] = null;
    return;
  }
  setOptionalNumber(target, key, value);
}

function validatePositiveInteger(errors: string[], label: string, value: string) {
  if (!value.trim()) return;
  const number = Number(value);
  if (!Number.isInteger(number) || number < 1) errors.push(`${label} must be a positive integer`);
}

function validateRequiredPositiveInteger(errors: string[], label: string, value: string) {
  if (!value.trim()) {
    errors.push(`${label} is required`);
    return;
  }
  validatePositiveInteger(errors, label, value);
}

function validatePriority(errors: string[], name: PhaseName, phase: ILMPhaseWizard) {
  if (!phase.setPriority.enabled) return;
  const value = phase.setPriority.priority.trim();
  if (!value) {
    errors.push(`${name} priority is required`);
    return;
  }
  if (value === 'null') return;
  const priority = Number(value);
  if (!Number.isInteger(priority) || priority < 0) errors.push(`${name} priority must be zero, a positive integer or null`);
}

function validateRolloverNumbers(errors: string[], name: PhaseName, phase: ILMPhaseWizard) {
  if (!phase.rollover.enabled) return;
  validatePositiveInteger(errors, `${name} rollover max documents`, phase.rollover.maxDocs);
  validatePositiveInteger(errors, `${name} rollover max primary shard documents`, phase.rollover.maxPrimaryShardDocs);
  validateNonNegativeInteger(errors, `${name} rollover min documents`, phase.rollover.minDocs);
  validateNonNegativeInteger(errors, `${name} rollover min primary shard documents`, phase.rollover.minPrimaryShardDocs);
}

function validateNonNegativeInteger(errors: string[], label: string, value: string) {
  if (!value.trim()) return;
  const number = Number(value);
  if (!Number.isInteger(number) || number < 0) errors.push(`${label} must be zero or a positive integer`);
}

function validateShrink(errors: string[], name: PhaseName, phase: ILMPhaseWizard) {
  if (!phase.shrink.enabled) return;
  const numberOfShards = phase.shrink.numberOfShards.trim();
  const maxPrimaryShardSize = phase.shrink.maxPrimaryShardSize.trim();
  if (Boolean(numberOfShards) === Boolean(maxPrimaryShardSize)) {
    errors.push(`${name} shrink requires exactly one target: number of shards or max primary shard size`);
    return;
  }
  validatePositiveInteger(errors, `${name} shrink number of shards`, numberOfShards);
}

function validateAllocate(errors: string[], name: PhaseName, phase: ILMPhaseWizard) {
  if (!phase.allocate.enabled) return;
  const hasRules = [...phase.allocate.include, ...phase.allocate.exclude, ...phase.allocate.require]
    .some((rule) => rule.attribute.trim());
  if (!phase.allocate.numberOfReplicas.trim() && !phase.allocate.totalShardsPerNode.trim() && !hasRules) {
    errors.push(`${name} allocate requires replicas, shards per node or an allocation rule`);
  }
  if (phase.allocate.numberOfReplicas.trim()) {
    const replicas = Number(phase.allocate.numberOfReplicas);
    if (!Number.isInteger(replicas) || replicas < 0) errors.push(`${name} replicas must be zero or a positive integer`);
  }
  if (phase.allocate.totalShardsPerNode.trim()) {
    const limit = Number(phase.allocate.totalShardsPerNode);
    if (!Number.isInteger(limit) || (limit < 1 && limit !== -1)) errors.push(`${name} shards per node must be a positive integer or -1`);
  }
}

function hasRolloverMaximum(rollover: ILMPhaseWizard['rollover']) {
  return Boolean(
    rollover.maxAge.trim()
    || rollover.maxDocs.trim()
    || rollover.maxPrimaryShardDocs.trim()
    || rollover.maxPrimaryShardSize.trim()
    || rollover.maxSize.trim(),
  );
}

function versionAtLeast(version: string | undefined, requiredMajor: number, requiredMinor: number) {
  if (!version) return true;
  const match = /^(\d+)(?:\.(\d+))?/.exec(version.trim());
  if (!match) return true;
  const major = Number(match[1]);
  const minor = Number(match[2] ?? 0);
  return major > requiredMajor || (major === requiredMajor && minor >= requiredMinor);
}

function parseRecord(value: string): RecordValue {
  try {
    return objectValue(JSON.parse(value));
  } catch {
    return {};
  }
}

function objectValue(value: unknown): RecordValue {
  return value && typeof value === 'object' && !Array.isArray(value) ? value as RecordValue : {};
}

function cloneRecord(value: RecordValue): RecordValue {
  return structuredClone(value);
}

function hasOwn(value: RecordValue, key: string) {
  return Object.prototype.hasOwnProperty.call(value, key);
}

function text(value: unknown) {
  if (value === null) return 'null';
  if (value === undefined) return '';
  return typeof value === 'string' ? value : String(value);
}
