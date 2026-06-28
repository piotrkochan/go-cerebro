import { useEffect, useState } from 'react';
import { useForm } from '@tanstack/react-form';

import { templatesCreate, templatesDelete, templatesList, type HostBodyWritable, type Template } from '../api/client';
import { DataTable, SortHeader, type DataTableColumn } from '../components/DataTable';
import { Icon } from '../components/Icon';
import { LazyJsonEditor } from '../components/LazyJsonEditor';
import { ConfirmModal } from '../components/Modal';
import { SplitPane } from '../components/SplitPane';
import { templateFormDefaults, type TemplateFormValues } from '../forms/templateForm';
import type { Notify } from '../types';
import { errorMessage, formatJson, parseJson, textValue } from '../utils/format';
import { nextSort, sortByText, type SortState } from '../utils/sort';

type TemplateSortKey = 'name' | 'pattern';

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
  const [deleteTemplate, setDeleteTemplate] = useState<Template | null>(null);
  const [sort, setSort] = useState<SortState<TemplateSortKey>>({ key: 'name', order: 'asc' });
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

  const filtered = sortByText(
    templates.filter((template) => {
      const pattern = templatePattern(template);
      return template.name.includes(filter.name) && pattern.includes(filter.pattern);
    }),
    sort,
    templateSortValue,
  );
  const templateColumns: DataTableColumn<Template>[] = [
    {
      header: <SortHeader name="name" sort={sort} onSort={(name) => setSort((value) => nextSort(value, name))}>name</SortHeader>,
      key: 'name',
      render: (template) => <><Icon name="book" /> {template.name}</>,
    },
    {
      header: <SortHeader name="pattern" sort={sort} onSort={(name) => setSort((value) => nextSort(value, name))}>pattern</SortHeader>,
      key: 'pattern',
      render: (template) => templatePattern(template),
    },
    {
      className: 'text-right',
      header: 'actions',
      headerClassName: 'text-right',
      key: 'actions',
      render: (template) => (
        <span className="inline-flex items-center justify-end gap-[10px]">
          <button
            aria-label={`edit template ${template.name}`}
            className="btn btn-default btn-xs"
            title="edit template"
            type="button"
            onClick={() => {
              form.setFieldValue('name', template.name);
              form.setFieldValue('body', formatJson(template.template));
            }}
          >
            <Icon name="pencil" />
          </button>
          <button aria-label={`delete template ${template.name}`} className="btn btn-danger btn-xs" title="delete template" type="button" onClick={() => setDeleteTemplate(template)}>
            <Icon name="trash" />
          </button>
        </span>
      ),
    },
  ];

  return (
    <>
      {deleteTemplate ? (
        <ConfirmModal
          body={
            <>
              Delete template <strong>{deleteTemplate.name}</strong>? This operation cannot be undone.
            </>
          }
          confirmLabel={
            <>
              <Icon name="trash" /> delete template
            </>
          }
          onClose={() => setDeleteTemplate(null)}
          onConfirm={() => remove(deleteTemplate.name)}
          title="delete template"
        />
      ) : null}
      <SplitPane
        storageKey="cerebro.templatesSplitPercent"
        left={
          <>
            <h4>
              existing templates <small className="info-text">({filtered.length})</small>
            </h4>
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
                <DataTable columns={templateColumns} getRowKey={(template) => template.name} rows={filtered} />
              </div>
            </div>
          </>
        }
        right={
          <>
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
          </>
        }
      />
    </>
  );
}

function templatePattern(template: Template) {
  return textValue((template.template as { template?: unknown })?.template);
}

function templateSortValue(template: Template, key: TemplateSortKey) {
  return key === 'name' ? template.name : templatePattern(template);
}
