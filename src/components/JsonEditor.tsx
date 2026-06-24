import { json } from '@codemirror/lang-json';
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { EditorView } from '@codemirror/view';
import CodeMirror from '@uiw/react-codemirror';
import { tags } from '@lezer/highlight';

const cerebroTheme = EditorView.theme(
  {
    '&': {
      backgroundColor: '#434749',
      color: '#f0f0f0',
      fontFamily: 'Monaco, Menlo, Consolas, "Courier New", monospace',
      fontSize: '12px',
    },
    '.cm-editor': {
      backgroundColor: '#434749',
      color: '#f0f0f0',
    },
    '.cm-scroller': {
      backgroundColor: '#434749',
      color: '#f0f0f0',
      fontFamily: 'Monaco, Menlo, Consolas, "Courier New", monospace',
    },
    '.cm-content': {
      backgroundColor: '#434749',
      caretColor: '#f0f0f0',
    },
    '.cm-line': {
      color: '#f0f0f0',
    },
    '.cm-cursor': {
      borderLeftColor: '#f0f0f0',
    },
    '.cm-gutters': {
      backgroundColor: '#373a3c',
      borderRight: '1px solid #55595c',
      color: '#c0c0c0',
    },
    '.cm-activeLine, .cm-activeLineGutter': {
      backgroundColor: '#373a3c',
    },
    '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {
      backgroundColor: '#55595c',
    },
    '.cm-panels, .cm-search': {
      backgroundColor: '#373a3c',
      color: '#f0f0f0',
    },
    '.cm-tooltip': {
      backgroundColor: '#373a3c',
      borderColor: '#55595c',
      color: '#f0f0f0',
    },
    '.cm-diagnostic': {
      color: '#f0f0f0',
    },
  },
  { dark: true },
);

const cerebroHighlight = syntaxHighlighting(
  HighlightStyle.define([
    { tag: tags.string, color: '#1AC98E' },
    { tag: tags.bool, color: '#1CA8DD', fontWeight: 'bold' },
    { tag: tags.number, color: '#9F85FF' },
    { tag: tags.null, color: '#D0D0D0' },
    { tag: tags.propertyName, color: '#f0f0f0' },
    { tag: tags.comment, color: '#998', fontStyle: 'italic' },
  ]),
);

export function JsonEditor({
  height,
  onChange,
  readOnly = false,
  value,
}: {
  height: number;
  onChange: (value: string) => void;
  readOnly?: boolean;
  value: string;
}) {
  return (
    <CodeMirror
      basicSetup
      className="json-editor"
      extensions={[json(), cerebroHighlight]}
      height={`${height}px`}
      onChange={onChange}
      readOnly={readOnly}
      theme={cerebroTheme}
      value={value}
    />
  );
}
