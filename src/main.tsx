import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import App from './App';
import './tailwind.css';
import './styles.css';

function normalizeLegacyHashBang() {
  if (!window.location.hash.startsWith('#!')) return;

  const legacyPath = window.location.hash.slice(2);
  const nextPath = legacyPath.startsWith('/') ? legacyPath : `/${legacyPath}`;
  window.history.replaceState(
    null,
    '',
    `${window.location.pathname}${window.location.search}#${nextPath}`,
  );
}

normalizeLegacyHashBang();

createRoot(document.getElementById('root') as HTMLElement).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
