export type SettingScope = 'cluster' | 'index';
export type SettingInput =
  | { kind: 'boolean' }
  | { kind: 'choice-text'; options: string[]; placeholder?: string }
  | { kind: 'number'; placeholder?: string; step?: string }
  | { kind: 'select'; options: string[] }
  | { kind: 'size'; placeholder?: string }
  | { kind: 'text'; placeholder?: string }
  | { kind: 'time'; placeholder?: string };

type SettingDefinition = {
  maxMajor?: number;
  minMajor?: number;
  name: string;
  scope: SettingScope;
};

const settingDefinitions: SettingDefinition[] = [
  ...clusterSettings([
    'action.auto_create_index',
    'action.destructive_requires_name',
    'action.search.shard_count.limit',
    'cluster.blocks.read_only',
    'cluster.blocks.read_only_allow_delete',
    'cluster.indices.close.enable',
    'cluster.info.update.interval',
    'cluster.info.update.timeout',
    'cluster.max_shards_per_node',
    'cluster.max_shards_per_node.frozen',
    'cluster.max_voting_config_exclusions',
    'cluster.persistent_tasks.allocation.enable',
    'cluster.routing.allocation.allow_rebalance',
    'cluster.routing.allocation.awareness.attributes',
    'cluster.routing.allocation.awareness.force.<attribute>.values',
    'cluster.routing.allocation.balance.disk_usage',
    'cluster.routing.allocation.balance.index',
    'cluster.routing.allocation.balance.shard',
    'cluster.routing.allocation.balance.threshold',
    'cluster.routing.allocation.balance.write_load',
    'cluster.routing.allocation.cluster_concurrent_rebalance',
    'cluster.routing.allocation.disk.reroute_interval',
    'cluster.routing.allocation.disk.threshold_enabled',
    'cluster.routing.allocation.disk.watermark.flood_stage',
    'cluster.routing.allocation.disk.watermark.flood_stage.frozen',
    'cluster.routing.allocation.disk.watermark.flood_stage.frozen.max_headroom',
    'cluster.routing.allocation.disk.watermark.flood_stage.max_headroom',
    'cluster.routing.allocation.disk.watermark.high',
    'cluster.routing.allocation.disk.watermark.high.max_headroom',
    'cluster.routing.allocation.disk.watermark.low',
    'cluster.routing.allocation.disk.watermark.low.max_headroom',
    'cluster.routing.allocation.enable',
    'cluster.routing.allocation.exclude.<attribute>',
    'cluster.routing.allocation.include.<attribute>',
    'cluster.routing.allocation.node_concurrent_incoming_recoveries',
    'cluster.routing.allocation.node_concurrent_outgoing_recoveries',
    'cluster.routing.allocation.node_concurrent_recoveries',
    'cluster.routing.allocation.node_initial_primaries_recoveries',
    'cluster.routing.allocation.require.<attribute>',
    'cluster.routing.allocation.same_shard.host',
    'cluster.routing.allocation.snapshot.relocation_enabled',
    'cluster.routing.allocation.total_shards_per_node',
    'cluster.routing.rebalance.enable',
    'cluster.remote.<cluster_alias>.mode',
    'cluster.remote.<cluster_alias>.proxy_address',
    'cluster.remote.<cluster_alias>.seeds',
    'cluster.remote.<cluster_alias>.skip_unavailable',
    'cluster.service.slow_task_logging_threshold',
    'indices.breaker.fielddata.limit',
    'indices.breaker.fielddata.overhead',
    'indices.breaker.request.limit',
    'indices.breaker.request.overhead',
    'indices.breaker.total.limit',
    'indices.breaker.total.use_real_memory',
    'indices.id_field_data.enabled',
    'indices.lifecycle.history_index_enabled',
    'indices.lifecycle.poll_interval',
    'indices.mapping.dynamic_timeout',
    'indices.recovery.max_bytes_per_sec',
    'ingest.geoip.downloader.eager.download',
    'ingest.geoip.downloader.enabled',
    'network.breaker.inflight_requests.limit',
    'network.breaker.inflight_requests.overhead',
    'script.max_compilations_rate',
    'search.allow_expensive_queries',
    'search.default_allow_partial_results',
    'search.default_keep_alive',
    'search.default_search_timeout',
    'search.low_level_cancellation',
    'search.max_buckets',
    'search.max_keep_alive',
    'search.max_open_scroll_context',
    'transport.tracer.exclude.<pattern>',
    'xpack.ml.node_concurrent_job_allocations',
    'xpack.monitoring.collection.enabled',
    'xpack.monitoring.collection.cluster.state.timeout',
    'xpack.monitoring.collection.cluster.stats.timeout',
    'xpack.monitoring.collection.index.recovery.active_only',
    'xpack.monitoring.collection.index.recovery.timeout',
    'xpack.monitoring.collection.index.stats.timeout',
    'xpack.monitoring.collection.interval',
    'xpack.monitoring.collection.ml.job.stats.timeout',
    'xpack.monitoring.history.duration',
    'xpack.security.http.filter.enabled',
    'xpack.security.transport.filter.enabled',
    'xpack.watcher.history.cleaner_service.enabled',
  ]),
  ...clusterSettings([
    'discovery.zen.commit_timeout',
    'discovery.zen.minimum_master_nodes',
    'discovery.zen.no_master_block',
    'discovery.zen.publish_diff.enable',
    'discovery.zen.publish_timeout',
    'gateway.initial_shards',
    'indices.recovery.internal_action_long_timeout',
    'indices.recovery.internal_action_timeout',
    'indices.recovery.recovery_activity_timeout',
    'indices.recovery.retry_delay_network',
    'indices.recovery.retry_delay_state_sync',
    'indices.store.throttle.max_bytes_per_sec',
    'indices.store.throttle.type',
    'indices.ttl.interval',
    'ingest.new_date_format',
    'script.max_compilations_per_minute',
  ], { maxMajor: 6 }),
  ...indexSettings([
    'index.allocation.max_retries',
    'index.auto_expand_replicas',
    'index.blocks.metadata',
    'index.blocks.read',
    'index.blocks.read_only',
    'index.blocks.read_only_allow_delete',
    'index.blocks.write',
    'index.default_pipeline',
    'index.final_pipeline',
    'index.gc_deletes',
    'index.highlight.max_analyzed_offset',
    'index.indexing.slowlog.include.user',
    'index.indexing.slowlog.level',
    'index.indexing.slowlog.reformat',
    'index.indexing.slowlog.source',
    'index.indexing.slowlog.threshold.index.debug',
    'index.indexing.slowlog.threshold.index.info',
    'index.indexing.slowlog.threshold.index.trace',
    'index.indexing.slowlog.threshold.index.warn',
    'index.lifecycle.indexing_complete',
    'index.lifecycle.name',
    'index.lifecycle.origination_date',
    'index.lifecycle.parse_origination_date',
    'index.lifecycle.rollover_alias',
    'index.load_fixed_bitset_filters_eagerly',
    'index.mapping.coerce',
    'index.mapping.depth.limit',
    'index.mapping.dimension_fields.limit',
    'index.mapping.field_name_length.ignore_dynamic_beyond_limit',
    'index.mapping.field_name_length.limit',
    'index.mapping.ignore_malformed',
    'index.mapping.nested_fields.limit',
    'index.mapping.nested_objects.limit',
    'index.mapping.total_fields.ignore_dynamic_beyond_limit',
    'index.mapping.total_fields.limit',
    'index.max_adjacency_matrix_filters',
    'index.max_docvalue_fields_search',
    'index.max_inner_result_window',
    'index.max_ngram_diff',
    'index.max_refresh_listeners',
    'index.max_regex_length',
    'index.max_rescore_window',
    'index.max_result_window',
    'index.max_script_fields',
    'index.max_shingle_diff',
    'index.max_slices_per_scroll',
    'index.max_terms_count',
    'index.merge.policy.expunge_deletes_allowed',
    'index.merge.policy.floor_segment',
    'index.merge.policy.max_merge_at_once',
    'index.merge.policy.max_merge_at_once_explicit',
    'index.merge.policy.max_merged_segment',
    'index.merge.policy.reclaim_deletes_weight',
    'index.merge.policy.segments_per_tier',
    'index.merge.scheduler.auto_throttle',
    'index.merge.scheduler.max_merge_count',
    'index.merge.scheduler.max_thread_count',
    'index.number_of_replicas',
    'index.priority',
    'index.queries.cache.enabled',
    'index.query.default_field',
    'index.query.parse.allow_unmapped_fields',
    'index.refresh_interval',
    'index.requests.cache.enable',
    'index.routing.allocation.enable',
    'index.routing.allocation.exclude.<attribute>',
    'index.routing.allocation.include.<attribute>',
    'index.routing.allocation.include._tier_preference',
    'index.routing.allocation.require.<attribute>',
    'index.routing.allocation.total_shards_per_node',
    'index.routing.rebalance.enable',
    'index.search.default_pipeline',
    'index.search.idle.after',
    'index.search.slowlog.include.user',
    'index.search.slowlog.level',
    'index.search.slowlog.threshold.fetch.debug',
    'index.search.slowlog.threshold.fetch.info',
    'index.search.slowlog.threshold.fetch.trace',
    'index.search.slowlog.threshold.fetch.warn',
    'index.search.slowlog.threshold.query.debug',
    'index.search.slowlog.threshold.query.info',
    'index.search.slowlog.threshold.query.trace',
    'index.search.slowlog.threshold.query.warn',
    'index.soft_deletes.retention_lease.period',
    'index.translog.durability',
    'index.translog.flush_threshold_size',
    'index.translog.sync_interval',
    'index.unassigned.node_left.delayed_timeout',
    'index.write.wait_for_active_shards',
  ]),
  ...indexSettings([
    'index.translog.retention.age',
    'index.translog.retention.size',
  ], { maxMajor: 6 }),
];

export function settingSuggestions(scope: SettingScope, major?: number) {
  return settingDefinitions
    .filter((setting) => setting.scope === scope && supportsMajor(setting, major))
    .map((setting) => setting.name)
    .sort((a, b) => a.localeCompare(b));
}

export function isDynamicSetting(scope: SettingScope, name: string, major?: number) {
  return settingDefinitions.some((setting) => setting.scope === scope && supportsMajor(setting, major) && matchesSetting(setting.name, name));
}

export function settingInput(scope: SettingScope, name: string, major?: number): SettingInput {
  const explicit = explicitSettingInput(scope, name, major);
  if (explicit) return explicit;
  if (isBooleanSetting(name)) return { kind: 'boolean' };
  if (isTimeSetting(name)) return { kind: 'time' };
  if (isSizeSetting(name)) return { kind: 'size' };
  if (isNumberSetting(name)) return { kind: 'number', step: isIntegerSetting(name) ? '1' : 'any' };
  return { kind: 'text' };
}

export function normalizeSettingValue(scope: SettingScope, name: string, value: string, major?: number) {
  if (settingInput(scope, name, major).kind !== 'number') return value;
  return expandExponentialNumber(value);
}

export function parseMajorVersion(version: unknown): number | undefined {
  if (typeof version !== 'string') return undefined;
  const major = Number(version.split('.')[0]);
  return Number.isFinite(major) ? major : undefined;
}

export function majorFromIndexVersionCreated(value: string | undefined): number | undefined {
  if (!value) return undefined;
  const versionID = Number(value);
  if (!Number.isFinite(versionID) || versionID <= 0) return undefined;
  const major = Math.floor(versionID / 1_000_000);
  return major > 0 ? major : undefined;
}

function clusterSettings(names: string[], version: Pick<SettingDefinition, 'maxMajor' | 'minMajor'> = {}) {
  return names.map((name) => ({ ...version, name, scope: 'cluster' as const }));
}

function indexSettings(names: string[], version: Pick<SettingDefinition, 'maxMajor' | 'minMajor'> = {}) {
  return names.map((name) => ({ ...version, name, scope: 'index' as const }));
}

function supportsMajor(setting: SettingDefinition, major?: number) {
  if (!major) return true;
  if (setting.minMajor && major < setting.minMajor) return false;
  if (setting.maxMajor && major > setting.maxMajor) return false;
  return true;
}

function matchesSetting(pattern: string, name: string) {
  if (!pattern.includes('<')) return pattern === name;
  const escaped = pattern.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const expression = escaped.replace(/<[^>]+>/g, '[^.]+');
  return new RegExp(`^${expression}$`).test(name);
}

function expandExponentialNumber(value: string) {
  const trimmed = value.trim();
  const match = /^([+-]?)(\d+)(?:\.(\d+))?[eE]([+-]?\d+)$/.exec(trimmed);
  if (!match) return value;

  const [, sign, integer, fraction = '', exponentText] = match;
  const exponent = Number(exponentText);
  if (!Number.isSafeInteger(exponent)) return value;

  const digits = `${integer}${fraction}`;
  const decimalPosition = integer.length + exponent;
  if (decimalPosition <= 0) {
    return stripDecimalZeros(`${sign}0.${'0'.repeat(Math.abs(decimalPosition))}${digits}`);
  }
  if (decimalPosition >= digits.length) {
    return stripDecimalZeros(`${sign}${digits}${'0'.repeat(decimalPosition - digits.length)}`);
  }
  return stripDecimalZeros(`${sign}${digits.slice(0, decimalPosition)}.${digits.slice(decimalPosition)}`);
}

function stripDecimalZeros(value: string) {
  if (!value.includes('.')) return value;
  return value.replace(/0+$/, '').replace(/\.$/, '');
}

function explicitSettingInput(scope: SettingScope, name: string, major?: number): SettingInput | undefined {
  const input = explicitInputs[`${scope}:${name}`] ?? explicitInputs[name];
  if (input) return input;
  if (isDynamicSetting(scope, name, major) && name.startsWith('cluster.remote.') && name.endsWith('.mode')) {
    return select(['sniff', 'proxy']);
  }
  return undefined;
}

const explicitInputs: Record<string, SettingInput> = {
  'action.auto_create_index': choiceText(['true', 'false'], 'true, false, +logs-*,-tmp-*,*'),
  'action.search.shard_count.limit': integer(),
  'cluster.info.update.interval': time(),
  'cluster.info.update.timeout': time(),
  'cluster.max_shards_per_node': integer(),
  'cluster.max_shards_per_node.frozen': integer(),
  'cluster.max_voting_config_exclusions': integer(),
  'cluster.persistent_tasks.allocation.enable': select(['all', 'none']),
  'cluster.routing.allocation.awareness.attributes': text('rack_id,zone'),
  'cluster.routing.allocation.awareness.force.<attribute>.values': text('zone-a,zone-b'),
  'cluster.routing.allocation.balance.disk_usage': decimal(),
  'cluster.routing.allocation.balance.index': decimal(),
  'cluster.routing.allocation.balance.shard': decimal(),
  'cluster.routing.allocation.balance.threshold': decimal(),
  'cluster.routing.allocation.balance.write_load': decimal(),
  'cluster.routing.allocation.cluster_concurrent_rebalance': integer(),
  'cluster.routing.allocation.disk.watermark.flood_stage': { kind: 'size' },
  'cluster.routing.allocation.disk.watermark.flood_stage.frozen': { kind: 'size' },
  'cluster.routing.allocation.disk.watermark.flood_stage.frozen.max_headroom': { kind: 'size' },
  'cluster.routing.allocation.disk.watermark.flood_stage.max_headroom': { kind: 'size' },
  'cluster.routing.allocation.disk.watermark.high': { kind: 'size' },
  'cluster.routing.allocation.disk.watermark.high.max_headroom': { kind: 'size' },
  'cluster.routing.allocation.disk.watermark.low': { kind: 'size' },
  'cluster.routing.allocation.disk.watermark.low.max_headroom': { kind: 'size' },
  'cluster.routing.allocation.allow_rebalance': select(['always', 'indices_primaries_active', 'indices_all_active']),
  'cluster.routing.allocation.enable': select(['all', 'primaries', 'new_primaries', 'none']),
  'cluster.routing.allocation.exclude.<attribute>': text('node-a,node-b'),
  'cluster.routing.allocation.include.<attribute>': text('node-a,node-b'),
  'cluster.routing.allocation.node_concurrent_incoming_recoveries': integer(),
  'cluster.routing.allocation.node_concurrent_outgoing_recoveries': integer(),
  'cluster.routing.allocation.node_concurrent_recoveries': integer(),
  'cluster.routing.allocation.node_initial_primaries_recoveries': integer(),
  'cluster.routing.allocation.require.<attribute>': text('node-a,node-b'),
  'cluster.routing.allocation.total_shards_per_node': integer(),
  'cluster.routing.rebalance.enable': select(['all', 'primaries', 'replicas', 'none']),
  'cluster.remote.<cluster_alias>.proxy_address': text('host:port'),
  'cluster.remote.<cluster_alias>.seeds': text('host1:9300,host2:9300'),
  'cluster.service.slow_task_logging_threshold': time(),
  'discovery.zen.no_master_block': select(['all', 'write']),
  'indices.breaker.fielddata.overhead': decimal(),
  'indices.breaker.fielddata.limit': { kind: 'size' },
  'indices.breaker.request.overhead': decimal(),
  'indices.breaker.request.limit': { kind: 'size' },
  'indices.breaker.total.overhead': decimal(),
  'indices.breaker.total.limit': { kind: 'size' },
  'indices.lifecycle.poll_interval': time(),
  'indices.mapping.dynamic_timeout': time(),
  'indices.recovery.max_bytes_per_sec': size('40mb'),
  'network.breaker.inflight_requests.limit': size(),
  'network.breaker.inflight_requests.overhead': decimal(),
  'script.max_compilations_per_minute': integer(),
  'script.max_compilations_rate': text('150/5m'),
  'search.default_keep_alive': time(),
  'search.default_search_timeout': time(),
  'search.max_buckets': integer(),
  'search.max_keep_alive': time(),
  'search.max_open_scroll_context': integer(),
  'transport.tracer.exclude.<pattern>': text('internal:coordination/*'),
  'xpack.ml.node_concurrent_job_allocations': integer(),
  'xpack.monitoring.collection.cluster.state.timeout': time(),
  'xpack.monitoring.collection.cluster.stats.timeout': time(),
  'xpack.monitoring.collection.index.recovery.timeout': time(),
  'xpack.monitoring.collection.index.stats.timeout': time(),
  'xpack.monitoring.collection.interval': time(),
  'xpack.monitoring.collection.ml.job.stats.timeout': time(),
  'xpack.monitoring.history.duration': time(),
  'gateway.initial_shards': select(['full', 'quorum', 'quorum-1', 'one']),
  'index.allocation.max_retries': integer(),
  'index.auto_expand_replicas': text('false, 0-1, 0-all'),
  'index.default_pipeline': text('pipeline-name'),
  'index.final_pipeline': text('pipeline-name'),
  'index.gc_deletes': time(),
  'index.highlight.max_analyzed_offset': integer(),
  'index.indexing.slowlog.level': select(['trace', 'debug', 'info', 'warn']),
  'index.indexing.slowlog.source': text('true, false, 1000'),
  'index.indexing.slowlog.threshold.index.debug': time(),
  'index.indexing.slowlog.threshold.index.info': time(),
  'index.indexing.slowlog.threshold.index.trace': time(),
  'index.indexing.slowlog.threshold.index.warn': time(),
  'index.lifecycle.name': text('policy-name'),
  'index.lifecycle.origination_date': integer('unix millis'),
  'index.lifecycle.rollover_alias': text('alias-name'),
  'index.mapping.depth.limit': integer(),
  'index.mapping.dimension_fields.limit': integer(),
  'index.mapping.field_name_length.limit': integer(),
  'index.mapping.nested_fields.limit': integer(),
  'index.mapping.nested_objects.limit': integer(),
  'index.mapping.total_fields.limit': integer(),
  'index.max_adjacency_matrix_filters': integer(),
  'index.max_docvalue_fields_search': integer(),
  'index.max_inner_result_window': integer(),
  'index.max_ngram_diff': integer(),
  'index.max_refresh_listeners': integer(),
  'index.max_regex_length': integer(),
  'index.max_rescore_window': integer(),
  'index.max_result_window': integer(),
  'index.max_script_fields': integer(),
  'index.max_shingle_diff': integer(),
  'index.max_slices_per_scroll': integer(),
  'index.max_terms_count': integer(),
  'index.merge.policy.expunge_deletes_allowed': decimal(),
  'index.merge.policy.floor_segment': size(),
  'index.merge.policy.max_merge_at_once': integer(),
  'index.merge.policy.max_merge_at_once_explicit': integer(),
  'index.merge.policy.max_merged_segment': size(),
  'index.merge.policy.reclaim_deletes_weight': decimal(),
  'index.merge.policy.segments_per_tier': decimal(),
  'index.merge.scheduler.max_merge_count': integer(),
  'index.merge.scheduler.max_thread_count': integer(),
  'index.number_of_replicas': integer(),
  'index.priority': integer(),
  'index.query.default_field': text('field1,field2,*'),
  'index.refresh_interval': time('1s, 30s, -1'),
  'index.routing.allocation.exclude.<attribute>': text('node-a,node-b'),
  'index.routing.allocation.include.<attribute>': text('node-a,node-b'),
  'index.routing.allocation.include._tier_preference': text('data_hot,data_warm'),
  'index.routing.allocation.require.<attribute>': text('node-a,node-b'),
  'index.routing.allocation.total_shards_per_node': integer(),
  'index.routing.allocation.enable': select(['all', 'primaries', 'new_primaries', 'none']),
  'index.routing.rebalance.enable': select(['all', 'primaries', 'replicas', 'none']),
  'index.search.default_pipeline': text('pipeline-name'),
  'index.search.idle.after': time(),
  'index.search.slowlog.level': select(['trace', 'debug', 'info', 'warn']),
  'index.search.slowlog.threshold.fetch.debug': time(),
  'index.search.slowlog.threshold.fetch.info': time(),
  'index.search.slowlog.threshold.fetch.trace': time(),
  'index.search.slowlog.threshold.fetch.warn': time(),
  'index.search.slowlog.threshold.query.debug': time(),
  'index.search.slowlog.threshold.query.info': time(),
  'index.search.slowlog.threshold.query.trace': time(),
  'index.search.slowlog.threshold.query.warn': time(),
  'index.soft_deletes.retention_lease.period': time(),
  'index.translog.flush_threshold_size': size(),
  'index.translog.retention.age': time(),
  'index.translog.retention.size': size(),
  'index.translog.sync_interval': time(),
  'index.translog.durability': select(['request', 'async']),
  'index.unassigned.node_left.delayed_timeout': time(),
  'index.write.wait_for_active_shards': text('all, 1, 2'),
  'indices.store.throttle.type': select(['none', 'merge', 'all']),
};

function select(options: string[]): SettingInput {
  return { kind: 'select', options };
}

function choiceText(options: string[], placeholder?: string): SettingInput {
  return { kind: 'choice-text', options, placeholder };
}

function decimal(placeholder?: string): SettingInput {
  return { kind: 'number', placeholder, step: 'any' };
}

function integer(placeholder?: string): SettingInput {
  return { kind: 'number', placeholder, step: '1' };
}

function size(placeholder = '512mb, 10gb, 40%'): SettingInput {
  return { kind: 'size', placeholder };
}

function text(placeholder?: string): SettingInput {
  return { kind: 'text', placeholder };
}

function time(placeholder = '30s, 5m, 1h'): SettingInput {
  return { kind: 'time', placeholder };
}

function isBooleanSetting(name: string) {
  return (
    name.endsWith('.enabled') ||
    name.endsWith('.enable') ||
    name.endsWith('.active_only') ||
    name.endsWith('.allow_unmapped_fields') ||
    name.endsWith('.coerce') ||
    name.endsWith('.download') ||
    name.endsWith('.filter.enabled') ||
    name.endsWith('.ignore_dynamic_beyond_limit') ||
    name.endsWith('.ignore_malformed') ||
    name.endsWith('.include.user') ||
    name.endsWith('.indexing_complete') ||
    name.endsWith('.metadata') ||
    name.endsWith('.parse_origination_date') ||
    name.endsWith('.read') ||
    name.endsWith('.read_only') ||
    name.endsWith('.read_only_allow_delete') ||
    name.endsWith('.reformat') ||
    name.endsWith('.skip_unavailable') ||
    name.endsWith('.threshold_enabled') ||
    name.endsWith('.use_real_memory') ||
    name.endsWith('.write') ||
    name === 'action.destructive_requires_name' ||
    name === 'cluster.routing.allocation.same_shard.host' ||
    name === 'indices.id_field_data.enabled' ||
    name === 'indices.lifecycle.history_index_enabled' ||
    name === 'search.allow_expensive_queries' ||
    name === 'search.default_allow_partial_results' ||
    name === 'search.low_level_cancellation'
  );
}

function isTimeSetting(name: string) {
  return (
    name.endsWith('.age') ||
    name.endsWith('.delay_network') ||
    name.endsWith('.delay_state_sync') ||
    name.endsWith('.delayed_timeout') ||
    name.endsWith('.duration') ||
    name.endsWith('.gc_deletes') ||
    name.endsWith('.interval') ||
    name.endsWith('.keep_alive') ||
    name.endsWith('.period') ||
    name.endsWith('.sync_interval') ||
    name.endsWith('.timeout') ||
    name.endsWith('.after') ||
    name.includes('.threshold.') ||
    name === 'cluster.service.slow_task_logging_threshold' ||
    name === 'search.default_search_timeout'
  );
}

function isSizeSetting(name: string) {
  return name.endsWith('.max_bytes_per_sec') || name.endsWith('.flush_threshold_size') || name.endsWith('.retention.size') || name.endsWith('.floor_segment') || name.endsWith('.max_merged_segment');
}

function isIntegerSetting(name: string) {
  return (
    name.includes('concurrent') ||
    name.includes('count') ||
    name.includes('limit') ||
    name.includes('max_') ||
    name.includes('number_of_') ||
    name.includes('priority') ||
    name.includes('retries') ||
    name.includes('shards_per_node')
  );
}

function isNumberSetting(name: string) {
  return isIntegerSetting(name) || name.includes('.balance.') || name.endsWith('.overhead') || name.endsWith('.weight') || name.endsWith('.expunge_deletes_allowed') || name.endsWith('.segments_per_tier');
}
