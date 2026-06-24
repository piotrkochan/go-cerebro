import { useEffect, useState } from 'react';
import { useForm } from '@tanstack/react-form';

import { templatesCreate, templatesDelete, templatesList, type HostBodyWritable, type Template } from '../api/client';
import { Icon } from '../components/Icon';
import { LazyJsonEditor } from '../components/LazyJsonEditor';
import { templateFormDefaults, type TemplateFormValues } from '../forms/templateForm';
import type { Notify } from '../types';
import { errorMessage, formatJson, parseJson, textValue } from '../utils/format';

const templateBase = formatJson({
  aliases: {},
  mappings: {},
  settings: {},
  template: 'template pattern(e.g.: index_name_*)',
});

export function TemplatesPage({
  connection,
  notify,
  refreshTick,
}: {
  connection: HostBodyWritable;
  notify: Notify;
  refreshTick: number;
}) {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [filter, setFilter] = useState({ name: '', pattern: '' });
  const form = useForm({
    defaultValues: templateFormDefaults(templateBase),
    onSubmit: async ({ value }) => {
      await save(value);
    },
  });

  useEffect(() => {
    void loadTemplates();
  }, [connection, refreshTick]);

  async function loadTemplates() {
    try {
      const result = await templatesList<true>({ body: connection, throwOnError: true });
      setTemplates((result.data.items ?? []).sort((left, right) => left.name.localeCompare(right.name)));
    } catch (error) {
      notify('danger', `Error while loading templates: ${errorMessage(error)}`);
    }
  }

  async function save(values: TemplateFormValues) {
    try {
      await templatesCreate<true>({
        body: { ...connection, name: values.name, template: parseJson(values.body) ?? {} },
        throwOnError: true,
      });
      notify('info', templates.some((template) => template.name === values.name) ? 'Template successfully updated' : 'Template successfully created');
      await loadTemplates();
    } catch (error) {
      notify('danger', `Error creating template: ${errorMessage(error)}`);
    }
  }

  async function remove(templateName: string) {
    try {
      await templatesDelete<true>({ body: { ...connection, name: templateName }, throwOnError: true });
      notify('info', 'Template successfully deleted');
      await loadTemplates();
    } catch (error) {
      notify('danger', `Error deleting template: ${errorMessage(error)}`);
    }
  }

  const filtered = templates.filter((template) => {
    const pattern = textValue((template.template as { template?: unknown })?.template);
    return template.name.includes(filter.name) && pattern.includes(filter.pattern);
  });

  return (
    <div className="row">
      <div className="col-md-6">
        <h4>existing templates</h4>
        <div className="row">
          <div className="col-md-8">
            {templates.length ? (
              <div className="row">
                <div className="col-md-6 form-group">
                  <input className="form-control" placeholder="template name" value={filter.name} onChange={(event) => setFilter((value) => ({ ...value, name: event.target.value }))} />
                </div>
                <div className="col-md-6 form-group">
                  <input className="form-control" placeholder="template pattern" value={filter.pattern} onChange={(event) => setFilter((value) => ({ ...value, pattern: event.target.value }))} />
                </div>
              </div>
            ) : null}
          </div>
          <div className="col-xs-12">
            <table className="table">
              <tbody>
                {filtered.map((template) => (
                  <tr key={template.name}>
                    <td>
                      <Icon name="book" /> {template.name} <span className="info-text"> with pattern</span>{' '}
                      {textValue((template.template as { template?: unknown })?.template)}
                      <Icon className="normal-action alert-danger pull-right" name="trash" onClick={() => void remove(template.name)} />
                      <Icon className="normal-action pull-right" name="pencil" onClick={() => {
                        form.setFieldValue('name', template.name);
                        form.setFieldValue('body', formatJson(template.template));
                      }} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
      <div className="col-md-6">
        <form.Subscribe selector={(state) => state.values.name}>
          {(name) => <h4>{templates.some((template) => template.name === name) ? `update template ${name}` : 'create new template'}</h4>}
        </form.Subscribe>
        <div className="row">
          <div className="col-xs-12">
            <div className="form-group">
              <form.Field name="name">
                {(field) => <input className="form-control" placeholder="name" value={field.state.value} onChange={(event) => field.handleChange(event.target.value)} />}
              </form.Field>
            </div>
          </div>
          <div className="col-xs-12">
            <div className="form-group">
              <form.Field name="body">
                {(field) => <LazyJsonEditor height={600} value={field.state.value} onChange={field.handleChange} />}
              </form.Field>
            </div>
          </div>
          <div className="col-xs-12 text-right">
            <form.Subscribe selector={(state) => state.values.name}>
              {(name) => {
                const editMode = templates.some((template) => template.name === name);
                return (
                  <button className={`btn ${editMode ? 'btn-warning' : 'btn-success'}`} type="submit" onClick={() => void form.handleSubmit()}>
                    <Icon className={`${editMode ? 'alert-warning' : 'alert-success'} normal-action`} name={editMode ? 'save' : 'file'} />{' '}
                    {editMode ? 'update' : 'create'}
                  </button>
                );
              }}
            </form.Subscribe>
          </div>
        </div>
      </div>
    </div>
  );
}
