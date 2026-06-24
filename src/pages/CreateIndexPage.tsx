import { useEffect, useState } from 'react';
import { Link } from '@tanstack/react-router';
import { useForm } from '@tanstack/react-form';

import {
  commonsIndices,
  createIndexCreate,
  createIndexGetMetadata,
  type HostBodyWritable,
} from '../api/client';
import { Icon } from '../components/Icon';
import { LazyJsonEditor } from '../components/LazyJsonEditor';
import { createIndexFormDefaults, type CreateIndexFormValues } from '../forms/createIndexForm';
import type { Notify } from '../types';
import { errorMessage, formatJson } from '../utils/format';

export function CreateIndexPage({
  connection,
  notify,
  refreshTick,
}: {
  connection: HostBodyWritable;
  notify: Notify;
  refreshTick: number;
}) {
  const [indices, setIndices] = useState<string[]>([]);
  const form = useForm({
    defaultValues: createIndexFormDefaults,
    onSubmit: async ({ value }) => {
      await createIndex(value);
    },
  });

  useEffect(() => {
    let ignore = false;

    async function load() {
      try {
        const result = await commonsIndices<true>({ body: connection, throwOnError: true });
        if (!ignore) setIndices((result.data.items ?? []).sort((left, right) => left.localeCompare(right)));
      } catch (error) {
        notify('danger', errorMessage(error));
      }
    }

    void load();
    return () => {
      ignore = true;
    };
  }, [connection, notify, refreshTick]);

  async function loadIndexMetadata(index: string) {
    form.setFieldValue('sourceIndex', index);
    if (!index) return;

    try {
      const result = await createIndexGetMetadata<true>({
        body: { ...connection, index },
        throwOnError: true,
      });
      form.setFieldValue('settings', formatJson({ settings: result.data.settings, mappings: result.data.mappings }));
    } catch (error) {
      notify('danger', `Error while loading index settings: ${errorMessage(error)}`);
    }
  }

  async function createIndex(values: CreateIndexFormValues) {
    if (!values.name.trim()) {
      notify('danger', 'You must specify a valid index name');
      return;
    }

    let metadata: unknown;
    if (values.settings.trim()) {
      try {
        metadata = JSON.parse(values.settings);
      } catch (error) {
        notify('danger', `Malformed settings: ${errorMessage(error)}`);
        return;
      }
    } else {
      const indexSettings: Record<string, string> = {};
      if (values.shards.trim()) indexSettings.number_of_shards = values.shards.trim();
      if (values.replicas.trim()) indexSettings.number_of_replicas = values.replicas.trim();
      metadata = { settings: { index: indexSettings } };
    }

    try {
      await createIndexCreate<true>({
        body: { ...connection, index: values.name.trim(), metadata },
        throwOnError: true,
      });
      notify('success', 'Index successfully created');
    } catch (error) {
      notify('danger', `Error while creating index: ${errorMessage(error)}`);
    }
  }

  return (
    <div>
      <div className="content-panel">
        <div className="row">
          <div className="col-sm-6">
            <div className="row">
              <div className="col-xs-12">
                <div className="form-group">
                  <label className="form-label">name</label>
                  <form.Field name="name">
                    {(field) => (
                      <input
                        className="form-control"
                        placeholder="index name"
                        type="text"
                        value={field.state.value}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                    )}
                  </form.Field>
                </div>
              </div>
            </div>
            <div className="row">
              <div className="col-sm-6">
                <div className="form-group">
                  <label className="form-label">number of shards</label>
                  <form.Field name="shards">
                    {(field) => (
                      <input
                        className="form-control"
                        placeholder="# of shards"
                        type="number"
                        value={field.state.value}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                    )}
                  </form.Field>
                </div>
              </div>
              <div className="col-sm-6">
                <div className="form-group">
                  <label className="form-label">number of replicas</label>
                  <form.Field name="replicas">
                    {(field) => (
                      <input
                        className="form-control"
                        placeholder="# of replicas"
                        type="number"
                        value={field.state.value}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                    )}
                  </form.Field>
                </div>
              </div>
            </div>
            <div className="row">
              <div className="col-xs-12">
                <div className="form-group">
                  <label className="form-label">load settings from existing index</label>
                  <form.Field name="sourceIndex">
                    {(field) => (
                      <select
                        className="form-control"
                        value={field.state.value}
                        onChange={(event) => void loadIndexMetadata(event.target.value)}
                      >
                        <option value="" />
                        {indices.map((index) => (
                          <option key={index}>{index}</option>
                        ))}
                      </select>
                    )}
                  </form.Field>
                </div>
              </div>
            </div>
          </div>
          <div className="col-sm-6">
            <div className="form-group">
              <label className="form-label">settings</label>
              <form.Field name="settings">
                {(field) => <LazyJsonEditor height={600} value={field.state.value} onChange={field.handleChange} />}
              </form.Field>
            </div>
          </div>
        </div>
        <div className="row">
          <div className="col-lg-12">
            <span className="pull-right">
              <Link className="btn btn-default" to="/overview">
                back
              </Link>{' '}
              <button className="btn btn-primary" type="button" onClick={() => void form.handleSubmit()}>
                <Icon name="check" /> Create
              </button>
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
