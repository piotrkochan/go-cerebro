import { useEffect, useState } from 'react';

import {
  analysisAnalyzers,
  analysisAnalyzeAnalyzer,
  analysisAnalyzeField,
  analysisFields,
  analysisIndices,
  type HostBodyWritable,
} from '../api/client';
import { Icon } from '../components/Icon';
import type { Notify } from '../types';
import { errorMessage, textValue } from '../utils/format';
import { nextSort, sortByText, type SortState } from '../utils/sort';

type Token = { end_offset?: unknown; position?: unknown; start_offset?: unknown; token?: unknown };
type TokenSortKey = 'token' | 'position' | 'start_offset' | 'end_offset';

export function AnalysisPage({
  connection,
  notify,
  refreshTick,
}: {
  connection: HostBodyWritable;
  notify: Notify;
  refreshTick: number;
}) {
  const [indices, setIndices] = useState<string[]>([]);
  const [fields, setFields] = useState<string[]>([]);
  const [analyzers, setAnalyzers] = useState<string[]>([]);
  const [fieldIndex, setFieldIndex] = useState('');
  const [field, setField] = useState('');
  const [fieldText, setFieldText] = useState('');
  const [fieldTokens, setFieldTokens] = useState<Token[]>();
  const [analyzerIndex, setAnalyzerIndex] = useState('');
  const [analyzer, setAnalyzer] = useState('');
  const [analyzerText, setAnalyzerText] = useState('');
  const [analyzerTokens, setAnalyzerTokens] = useState<Token[]>();

  useEffect(() => {
    async function load() {
      try {
        const result = await analysisIndices<true>({ body: connection, throwOnError: true });
        setIndices((result.data.items ?? []).sort());
      } catch (error) {
        notify('danger', `Error loading indices: ${errorMessage(error)}`);
      }
    }
    void load();
  }, [connection, notify, refreshTick]);

  async function loadFields(index: string) {
    setFieldIndex(index);
    setField('');
    if (!index) return setFields([]);
    try {
      const result = await analysisFields<true>({ body: { ...connection, index }, throwOnError: true });
      setFields((result.data.items ?? []).sort());
    } catch (error) {
      setFields([]);
      notify('danger', `Error loading index fields: ${errorMessage(error)}`);
    }
  }

  async function loadAnalyzers(index: string) {
    setAnalyzerIndex(index);
    setAnalyzer('');
    if (!index) return setAnalyzers([]);
    try {
      const result = await analysisAnalyzers<true>({ body: { ...connection, index }, throwOnError: true });
      setAnalyzers((result.data.items ?? []).sort());
    } catch (error) {
      setAnalyzers([]);
      notify('danger', `Error loading index analyzers: ${errorMessage(error)}`);
    }
  }

  async function analyzeField() {
    if (!fieldIndex || !field || !fieldText) return;
    try {
      const result = await analysisAnalyzeField<true>({
        body: { ...connection, field, index: fieldIndex, text: fieldText },
        throwOnError: true,
      });
      setFieldTokens(tokens(result.data.data));
    } catch (error) {
      notify('danger', `Error analyzing text by field: ${errorMessage(error)}`);
    }
  }

  async function analyzeAnalyzer() {
    if (!analyzerIndex || !analyzer || !analyzerText) return;
    try {
      const result = await analysisAnalyzeAnalyzer<true>({
        body: { ...connection, analyzer, index: analyzerIndex, text: analyzerText },
        throwOnError: true,
      });
      setAnalyzerTokens(tokens(result.data.data));
    } catch (error) {
      notify('danger', `Error analyzing text by analyzer: ${errorMessage(error)}`);
    }
  }

  return (
    <div className="row">
      <div className="col-md-6">
        <h4>analyze by field type</h4>
        <div className="row">
          <Select label="index name" onChange={(value) => void loadFields(value)} options={indices} value={fieldIndex} />
          <Select label="field name" onChange={setField} options={fields} value={field} />
        </div>
        <TextInput value={fieldText} onChange={setFieldText} />
        <ActionButton onClick={() => void analyzeField()} />
        <TokensTable tokens={fieldTokens} />
      </div>
      <div className="col-md-6">
        <h4>analyze by analyzer</h4>
        <div className="row">
          <Select label="index name" onChange={(value) => void loadAnalyzers(value)} options={indices} value={analyzerIndex} />
          <Select label="analyzer name" onChange={setAnalyzer} options={analyzers} value={analyzer} />
        </div>
        <TextInput value={analyzerText} onChange={setAnalyzerText} />
        <ActionButton onClick={() => void analyzeAnalyzer()} />
        <TokensTable tokens={analyzerTokens} />
      </div>
    </div>
  );
}

function Select({ label, onChange, options, value }: { label: string; onChange: (value: string) => void; options: string[]; value: string }) {
  return (
    <div className="col-sm-4">
      <div className="form-group">
        <label className="form-label">{label}</label>
        <select className="form-control" value={value} onChange={(event) => onChange(event.target.value)}>
          <option value="" />
          {options.map((option) => <option key={option}>{option}</option>)}
        </select>
      </div>
    </div>
  );
}

function TextInput({ onChange, value }: { onChange: (value: string) => void; value: string }) {
  return (
    <div className="row">
      <div className="col-sm-12">
        <div className="form-group">
          <label className="form-label">&nbsp;</label>
          <input className="form-control" placeholder="text to analyze" type="text" value={value} onChange={(event) => onChange(event.target.value)} />
        </div>
      </div>
    </div>
  );
}

function ActionButton({ onClick }: { onClick: () => void }) {
  return (
    <div className="row">
      <div className="col-xs-12 text-right">
        <button className="btn btn-success" type="submit" onClick={onClick}>
          <Icon name="bolt" /> analyze
        </button>
      </div>
    </div>
  );
}

function TokensTable({ tokens }: { tokens?: Token[] }) {
  const [sort, setSort] = useState<SortState<TokenSortKey>>({ key: 'position', order: 'asc' });

  if (!tokens) return null;
  const sortedTokens = sortByText(tokens, sort, tokenSortValue);

  return (
    <table className="table">
      <thead>
        <tr>
          <th>
            <button className="normal-action border-0 bg-transparent p-0 text-inherit" type="button" onClick={() => setSort((value) => nextSort(value, 'token'))}>
              token {sort.key === 'token' ? <Icon name={sort.order === 'asc' ? 'caret-down' : 'sort-alpha-desc'} /> : null}
            </button>
          </th>
          <th>
            <button className="normal-action border-0 bg-transparent p-0 text-inherit" type="button" onClick={() => setSort((value) => nextSort(value, 'position'))}>
              position {sort.key === 'position' ? <Icon name={sort.order === 'asc' ? 'caret-down' : 'sort-alpha-desc'} /> : null}
            </button>
          </th>
          <th>
            <button className="normal-action border-0 bg-transparent p-0 text-inherit" type="button" onClick={() => setSort((value) => nextSort(value, 'start_offset'))}>
              start offset {sort.key === 'start_offset' ? <Icon name={sort.order === 'asc' ? 'caret-down' : 'sort-alpha-desc'} /> : null}
            </button>
          </th>
          <th>
            <button className="normal-action border-0 bg-transparent p-0 text-inherit" type="button" onClick={() => setSort((value) => nextSort(value, 'end_offset'))}>
              end offset {sort.key === 'end_offset' ? <Icon name={sort.order === 'asc' ? 'caret-down' : 'sort-alpha-desc'} /> : null}
            </button>
          </th>
        </tr>
      </thead>
      <tbody>
        {sortedTokens.map((token, index) => (
          <tr key={index}>
            <td>{textValue(token.token)}</td>
            <td>{textValue(token.position)}</td>
            <td>{textValue(token.start_offset)}</td>
            <td>{textValue(token.end_offset)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function tokenSortValue(token: Token, key: TokenSortKey) {
  return textValue(token[key]);
}

function tokens(data: unknown): Token[] {
  const obj = data as { tokens?: unknown };
  return Array.isArray(obj?.tokens) ? obj.tokens as Token[] : [];
}
