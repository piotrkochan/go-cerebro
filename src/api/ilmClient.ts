import { client } from './client/client.gen';
import type {
  HostBodyWritable,
  IlmPoliciesDeleteErrors,
  IlmPoliciesDeleteResponses,
  IlmPoliciesListErrors,
  IlmPoliciesListResponses,
  IlmPoliciesSaveErrors,
  IlmPoliciesSaveResponses,
  IlmPolicyNameInBodyWritable,
  IlmPolicySaveInBodyWritable,
} from './client/types.gen';

type JSONOptions<Body, ThrowOnError extends boolean> = {
  body: Body;
  headers?: Record<string, string>;
  throwOnError?: ThrowOnError;
};

const jsonHeaders = { 'Content-Type': 'application/json' };

export function ilmPoliciesList<ThrowOnError extends boolean = false>(options: JSONOptions<HostBodyWritable, ThrowOnError>) {
  return client.post<IlmPoliciesListResponses, IlmPoliciesListErrors, ThrowOnError>({
    url: '/ilm/policies',
    ...options,
    headers: { ...jsonHeaders, ...options.headers },
  });
}

export function ilmPoliciesSave<ThrowOnError extends boolean = false>(options: JSONOptions<IlmPolicySaveInBodyWritable, ThrowOnError>) {
  return client.post<IlmPoliciesSaveResponses, IlmPoliciesSaveErrors, ThrowOnError>({
    url: '/ilm/policies/save',
    ...options,
    headers: { ...jsonHeaders, ...options.headers },
  });
}

export function ilmPoliciesDelete<ThrowOnError extends boolean = false>(options: JSONOptions<IlmPolicyNameInBodyWritable, ThrowOnError>) {
  return client.post<IlmPoliciesDeleteResponses, IlmPoliciesDeleteErrors, ThrowOnError>({
    url: '/ilm/policies/delete',
    ...options,
    headers: { ...jsonHeaders, ...options.headers },
  });
}
