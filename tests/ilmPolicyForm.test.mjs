import assert from 'node:assert/strict';
import test from 'node:test';

import {
  createDefaultILMWizard,
  ilmFeaturesForVersion,
  policyBodyFromWizard,
  validateILMWizard,
  wizardFromPolicyBody,
} from '../src/forms/ilmPolicyForm.ts';

test('default wizard creates a hot rollover and delete policy', () => {
  const body = policyBodyFromWizard(createDefaultILMWizard(), '{}');

  assert.deepEqual(body, {
    policy: {
      phases: {
        delete: { actions: { delete: {} }, min_age: '90d' },
        hot: {
          actions: {
            rollover: { max_age: '30d', max_primary_shard_size: '50gb' },
          },
        },
      },
    },
  });
});

test('wizard edits preserve metadata and fields it does not manage', () => {
  const original = JSON.stringify({
    response_metadata: { owner: 'operations' },
    policy: {
      _meta: { description: 'keep me' },
      phases: {
        hot: {
          custom_phase_option: true,
          actions: {
            rollover: { max_age: '1d', future_option: 'keep me too' },
            future_action: { enabled: true },
          },
        },
        archive: { actions: { custom: {} } },
      },
    },
  });
  const wizard = wizardFromPolicyBody(original);
  wizard.hot.rollover.maxAge = '2d';

  const body = policyBodyFromWizard(wizard, original);

  assert.deepEqual(body.response_metadata, { owner: 'operations' });
  assert.deepEqual(body.policy._meta, { description: 'keep me' });
  assert.equal(body.policy.phases.hot.custom_phase_option, true);
  assert.deepEqual(body.policy.phases.hot.actions.future_action, { enabled: true });
  assert.equal(body.policy.phases.hot.actions.rollover.future_option, 'keep me too');
  assert.equal(body.policy.phases.hot.actions.rollover.max_age, '2d');
  assert.deepEqual(body.policy.phases.archive, { actions: { custom: {} } });
});

test('wizard serializes expanded actions with Elasticsearch field names', () => {
  const wizard = createDefaultILMWizard();
  wizard.hot.setPriority = { enabled: true, priority: '100' };
  wizard.hot.unfollow = true;
  wizard.hot.rollover.maxPrimaryShardDocs = '10000000';
  wizard.warm.enabled = true;
  wizard.warm.allocate = {
    enabled: true,
    exclude: [{ attribute: 'rack', value: 'old' }],
    include: [],
    numberOfReplicas: '1',
    require: [{ attribute: 'box_type', value: 'warm' }],
    totalShardsPerNode: '20',
  };
  wizard.warm.downsample.enabled = true;
  wizard.warm.downsample.fixedInterval = '1h';
  wizard.warm.downsample.samplingMethod = 'last_value';
  wizard.warm.forceMerge.enabled = true;
  wizard.warm.migrate = 'disabled';
  wizard.warm.shrink.enabled = true;
  wizard.cold.enabled = true;
  wizard.cold.searchableSnapshot = {
    enabled: true,
    forceMergeIndex: 'false',
    replicateFor: '7d',
    snapshotRepository: 'archive',
    totalShardsPerNode: '2',
  };
  wizard.frozen.enabled = true;
  wizard.frozen.searchableSnapshot.enabled = true;
  wizard.frozen.searchableSnapshot.snapshotRepository = 'deep-archive';
  wizard.delete.waitForSnapshot = { enabled: true, policy: 'daily-snapshots' };
  wizard.delete.delete.deleteSearchableSnapshot = 'false';

  const body = policyBodyFromWizard(wizard, '{}');
  const phases = body.policy.phases;

  assert.deepEqual(phases.hot.actions.set_priority, { priority: 100 });
  assert.deepEqual(phases.hot.actions.unfollow, {});
  assert.equal(phases.hot.actions.rollover.max_primary_shard_docs, 10000000);
  assert.deepEqual(phases.warm.actions.allocate, {
    exclude: { rack: 'old' },
    number_of_replicas: 1,
    require: { box_type: 'warm' },
    total_shards_per_node: 20,
  });
  assert.deepEqual(phases.warm.actions.downsample, { fixed_interval: '1h', sampling_method: 'last_value' });
  assert.deepEqual(phases.warm.actions.forcemerge, { max_num_segments: 1 });
  assert.deepEqual(phases.warm.actions.migrate, { enabled: false });
  assert.deepEqual(phases.warm.actions.shrink, { number_of_shards: 1 });
  assert.deepEqual(phases.cold.actions.searchable_snapshot, {
    force_merge_index: false,
    replicate_for: '7d',
    snapshot_repository: 'archive',
    total_shards_per_node: 2,
  });
  assert.equal(phases.frozen.actions.searchable_snapshot.snapshot_repository, 'deep-archive');
  assert.deepEqual(phases.delete.actions.wait_for_snapshot, { policy: 'daily-snapshots' });
  assert.deepEqual(phases.delete.actions.delete, { delete_searchable_snapshot: false });
});

test('wizard reads expanded action settings', () => {
  const wizard = wizardFromPolicyBody(JSON.stringify({
    policy: {
      phases: {
        warm: {
          min_age: '5d',
          actions: {
            allocate: { include: { rack: 'one,two' }, number_of_replicas: 2 },
            migrate: { enabled: false },
            set_priority: { priority: 25 },
          },
        },
      },
    },
  }));

  assert.equal(wizard.hot.enabled, false);
  assert.equal(wizard.warm.enabled, true);
  assert.equal(wizard.warm.minAge, '5d');
  assert.equal(wizard.warm.allocate.numberOfReplicas, '2');
  assert.deepEqual(wizard.warm.allocate.include, [{ attribute: 'rack', value: 'one,two' }]);
  assert.equal(wizard.warm.migrate, 'disabled');
  assert.deepEqual(wizard.warm.setPriority, { enabled: true, priority: '25' });
});

test('explicit default migrate action remains an empty action', () => {
  const original = JSON.stringify({ policy: { phases: { warm: { actions: { migrate: {} } } } } });
  const wizard = wizardFromPolicyBody(original);

  assert.deepEqual(policyBodyFromWizard(wizard, original).policy.phases.warm.actions.migrate, {});
});

test('feature availability follows the connected Elasticsearch version', () => {
  assert.deepEqual(ilmFeaturesForVersion('7.9.3'), {
    downsample: false,
    downsampleForceMerge: false,
    frozenPhase: false,
    migrate: false,
    searchableSnapshot: false,
    waitForSnapshot: true,
  });
  assert.equal(ilmFeaturesForVersion('7.10.0').searchableSnapshot, true);
  assert.equal(ilmFeaturesForVersion('8.6.2').downsample, false);
  assert.equal(ilmFeaturesForVersion('8.7.0').downsample, true);
  assert.equal(ilmFeaturesForVersion('9.2.3').downsampleForceMerge, false);
  assert.equal(ilmFeaturesForVersion('9.3.0').downsampleForceMerge, true);
  assert.equal(ilmFeaturesForVersion('9.5.3').downsample, true);
});

test('wizard validation rejects incomplete action configuration', () => {
  const wizard = createDefaultILMWizard();
  const features = ilmFeaturesForVersion('9.5.3');
  wizard.hot.rollover.maxAge = '';
  wizard.hot.rollover.maxPrimaryShardSize = '';
  wizard.warm.enabled = true;
  wizard.warm.allocate.enabled = true;
  wizard.cold.enabled = true;
  wizard.cold.searchableSnapshot.enabled = true;
  wizard.delete.waitForSnapshot.enabled = true;

  assert.deepEqual(validateILMWizard(wizard, features), [
    'warm allocate requires replicas, shards per node or an allocation rule',
    'cold searchable snapshot requires a repository',
    'hot rollover requires at least one maximum condition',
    'delete wait for snapshot requires an SLM policy',
  ]);
});

test('wizard validation rejects mixed downsample methods', () => {
  const wizard = createDefaultILMWizard();
  wizard.hot.downsample = { enabled: true, fixedInterval: '10m', forceMergeIndex: 'default', samplingMethod: 'aggregate' };
  wizard.warm.enabled = true;
  wizard.warm.downsample = { enabled: true, fixedInterval: '1h', forceMergeIndex: 'default', samplingMethod: 'last_value' };

  assert.deepEqual(validateILMWizard(wizard, ilmFeaturesForVersion('9.5.3')), [
    'all downsample actions must use the same sampling method',
  ]);
});

test('wizard validation checks required action numbers', () => {
  const wizard = createDefaultILMWizard();
  wizard.hot.forceMerge = { enabled: true, maxNumSegments: '' };
  wizard.hot.setPriority = { enabled: true, priority: '' };
  wizard.hot.rollover.maxDocs = '-1';

  assert.deepEqual(validateILMWizard(wizard, ilmFeaturesForVersion('9.5.3')), [
    'hot force merge max segments is required',
    'hot priority is required',
    'hot rollover max documents must be a positive integer',
  ]);
});
