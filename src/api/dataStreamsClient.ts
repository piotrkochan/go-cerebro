import { client } from './client/client.gen';
import type {
  DataStreamAttachIlmInBodyWritable,
  DataStreamDetachIlmInBodyWritable,
  DataStreamLifecycleInBodyWritable,
  DataStreamNameInBodyWritable,
  DataStreamsAttachIlmErrors,
  DataStreamsAttachIlmResponses,
  DataStreamsCreateErrors,
  DataStreamsCreateResponses,
  DataStreamsDeleteErrors,
  DataStreamsDeleteResponses,
  DataStreamsDetachIlmErrors,
  DataStreamsDetachIlmResponses,
  DataStreamsListErrors,
  DataStreamsListResponses,
  DataStreamsRolloverErrors,
  DataStreamsRolloverResponses,
  DataStreamsUpdateLifecycleErrors,
  DataStreamsUpdateLifecycleResponses,
  HostBodyWritable,
} from './client/types.gen';

type JSONOptions<Body, ThrowOnError extends boolean> = {
  body: Body;
  headers?: Record<string, string>;
  throwOnError?: ThrowOnError;
};

const jsonHeaders = { 'Content-Type': 'application/json' };

export function dataStreamsList<ThrowOnError extends boolean = false>(options: JSONOptions<HostBodyWritable, ThrowOnError>) {
  return client.post<DataStreamsListResponses, DataStreamsListErrors, ThrowOnError>({
    url: '/data_streams',
    ...options,
    headers: { ...jsonHeaders, ...options.headers },
  });
}

export function dataStreamsAttachIlm<ThrowOnError extends boolean = false>(options: JSONOptions<DataStreamAttachIlmInBodyWritable, ThrowOnError>) {
  return client.post<DataStreamsAttachIlmResponses, DataStreamsAttachIlmErrors, ThrowOnError>({
    url: '/data_streams/attach_ilm',
    ...options,
    headers: { ...jsonHeaders, ...options.headers },
  });
}

export function dataStreamsDetachIlm<ThrowOnError extends boolean = false>(options: JSONOptions<DataStreamDetachIlmInBodyWritable, ThrowOnError>) {
  return client.post<DataStreamsDetachIlmResponses, DataStreamsDetachIlmErrors, ThrowOnError>({
    url: '/data_streams/detach_ilm',
    ...options,
    headers: { ...jsonHeaders, ...options.headers },
  });
}

export function dataStreamsCreate<ThrowOnError extends boolean = false>(options: JSONOptions<DataStreamNameInBodyWritable, ThrowOnError>) {
  return client.post<DataStreamsCreateResponses, DataStreamsCreateErrors, ThrowOnError>({
    url: '/data_streams/create',
    ...options,
    headers: { ...jsonHeaders, ...options.headers },
  });
}

export function dataStreamsDelete<ThrowOnError extends boolean = false>(options: JSONOptions<DataStreamNameInBodyWritable, ThrowOnError>) {
  return client.post<DataStreamsDeleteResponses, DataStreamsDeleteErrors, ThrowOnError>({
    url: '/data_streams/delete',
    ...options,
    headers: { ...jsonHeaders, ...options.headers },
  });
}

export function dataStreamsRollover<ThrowOnError extends boolean = false>(options: JSONOptions<DataStreamNameInBodyWritable, ThrowOnError>) {
  return client.post<DataStreamsRolloverResponses, DataStreamsRolloverErrors, ThrowOnError>({
    url: '/data_streams/rollover',
    ...options,
    headers: { ...jsonHeaders, ...options.headers },
  });
}

export function dataStreamsUpdateLifecycle<ThrowOnError extends boolean = false>(options: JSONOptions<DataStreamLifecycleInBodyWritable, ThrowOnError>) {
  return client.post<DataStreamsUpdateLifecycleResponses, DataStreamsUpdateLifecycleErrors, ThrowOnError>({
    url: '/data_streams/lifecycle',
    ...options,
    headers: { ...jsonHeaders, ...options.headers },
  });
}
