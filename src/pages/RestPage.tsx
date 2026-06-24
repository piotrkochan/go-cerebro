import { useEffect, useState } from 'react';
import { useForm } from '@tanstack/react-form';

import { restHistory, restRequest, type HostBodyWritable } from '../api/client';
import { Icon } from '../components/Icon';
import { LazyJsonEditor } from '../components/LazyJsonEditor';
import { restRequestFormDefaults, type RestRequestFormValues } from '../forms/restRequestForm';
import { curl, formatJson, parseJson, textValue } from '../utils/format';

export function RestPage({ connection }: { connection: HostBodyWritable }) {
  const [response, setResponse] = useState<unknown>();
  const [history, setHistory] = useState<Array<{ created_at?: unknown; data?: unknown; method?: unknown; path?: unknown }>>([]);
  const form = useForm({
    defaultValues: restRequestFormDefaults,
    onSubmit: async ({ value }) => {
      await execute(value);
    },
  });

  useEffect(() => {
    async function load() {
      try {
        const result = await restHistory<true>({ body: connection, throwOnError: true });
        setHistory(Array.isArray(result.data.data) ? result.data.data : []);
      } catch {
        setHistory([]);
      }
    }
    void load();
  }, [connection]);

  async function execute(values: RestRequestFormValues) {
    const result = await restRequest<true>({
      body: { ...connection, data: parseJson(values.body), method: values.method, path: values.path },
      throwOnError: true,
    });
    setResponse(result.data.data);
  }

  function setRequest(values: RestRequestFormValues) {
    form.setFieldValue('method', values.method);
    form.setFieldValue('path', values.path);
    form.setFieldValue('body', values.body);
  }

  return (
    <div className="row">
      <div className="col-md-6">
        <div className="row">
          <div className="col-lg-10 col-md-9 typeahead-demo form-group">
            <form.Field name="path">
              {(field) => (
                <input
                  className="form-control form-control-sm"
                  placeholder="path (eg: indexName/_search)"
                  type="text"
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                />
              )}
            </form.Field>
          </div>
          <div className="col-lg-2 col-md-3 form-group">
            <form.Field name="method">
              {(field) => (
                <select className="form-control" value={field.state.value} onChange={(event) => field.handleChange(event.target.value)}>
                  {['GET', 'PUT', 'POST', 'DELETE'].map((value) => <option key={value}>{value}</option>)}
                </select>
              )}
            </form.Field>
          </div>
        </div>
        <div className="row">
          <div className="col-xs-12">
            <a href="#restHistory"><Icon name="history" /> previous requests</a>
          </div>
          {history.length ? (
            <div className="col-xs-12 panel-collapse collapse in !block !visible" id="restHistory">
              <table className="table table-condensed">
                <tbody>
                  {history.map((item, index) => (
                    <tr className="normal-action" key={index} onClick={() => {
                      setRequest({
                        body: formatJson(item.data ?? ''),
                        method: textValue(item.method) || 'GET',
                        path: textValue(item.path),
                      });
                    }}>
                      <td style={{ width: 100 }}>{textValue(item.created_at)}</td>
                      <td style={{ width: 60 }}>{textValue(item.method)}</td>
                      <td>{textValue(item.path)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : null}
        </div>
        <div className="form-group row">
          <div className="col-lg-12">
            <form.Field name="body">
              {(field) => <LazyJsonEditor height={600} value={field.state.value} onChange={field.handleChange} />}
            </form.Field>
          </div>
        </div>
        <div className="form-group row">
          <div className="col-lg-12 text-right">
            <div className="btn-group">
              <form.Subscribe selector={(state) => state.values}>
                {(values) => (
                  <button className="btn btn-default" type="button" onClick={() => void navigator.clipboard?.writeText(curl(values.method, values.path, values.body))}>
                    <Icon name="clipboard" /> cURL
                  </button>
                )}
              </form.Subscribe>
            </div>
            <div className="btn-group">
              <form.Subscribe selector={(state) => state.values.body}>
                {(body) => (
                  <button className="btn btn-default" type="button" onClick={() => form.setFieldValue('body', formatJson(parseJson(body) ?? {}))}>
                    <Icon name="align-left" /> format
                  </button>
                )}
              </form.Subscribe>
            </div>
            <div className="btn-group">
              <button className="btn btn-success" type="button" onClick={() => void form.handleSubmit()}>
                <Icon name="bolt" /> send
              </button>
            </div>
          </div>
        </div>
      </div>
      <div className="col-md-6">
        <div style={{ border: '1px solid #55595c', minHeight: 647, overflow: 'auto', display: 'block' }}>
          <div className="modal-body"><pre>{formatJson(response ?? '')}</pre></div>
        </div>
      </div>
    </div>
  );
}
