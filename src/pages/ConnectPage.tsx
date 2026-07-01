import { useEffect, useState } from 'react';
import { useForm } from '@tanstack/react-form';

import { connect, connectHosts, type HostRef } from '../api/client';
import { loadAuthStatus } from '../api/security';
import { Button } from '../components/Button';
import { CerebroLogo } from '../components/CerebroLogo';
import { Icon } from '../components/Icon';
import { connectFormDefaults } from '../forms/connectForm';
import { cleanConnection } from '../utils/connection';
import { errorMessage } from '../utils/format';
import { APP_VERSION } from '../version';

type AuthStatus = {
  authenticated: boolean;
  enabled: boolean;
  user: string;
};

export function ConnectPage({
  currentHost,
  onConnected,
}: {
  currentHost: string;
  onConnected: (host: string, hostName: string) => void;
}) {
  const [authStatus, setAuthStatus] = useState<AuthStatus | null>(null);
  const [hosts, setHosts] = useState<HostRef[]>([]);
  const [connecting, setConnecting] = useState(false);
  const [feedback, setFeedback] = useState('');
  const connectForm = useForm({
    defaultValues: connectFormDefaults(currentHost),
    onSubmit: async ({ value }) => {
      await submit(value.host);
    },
  });

  useEffect(() => {
    let ignore = false;
    async function syncAuthStatus() {
      const data = await loadAuthStatus();
      if (!ignore && data) {
        setAuthStatus({
          authenticated: data.authenticated === true,
          enabled: data.enabled === true,
          user: typeof data.user === 'string' ? data.user : '',
        });
      }
    }
    async function load() {
      try {
        const result = await connectHosts<true>({ throwOnError: true });
        if (!ignore) setHosts(result.data.items ?? []);
      } catch {
        if (!ignore) setHosts([]);
      }
    }
    void syncAuthStatus();
    void load();
    return () => {
      ignore = true;
    };
  }, []);

  useEffect(() => {
    connectForm.setFieldValue('host', currentHost);
  }, [connectForm, currentHost]);

  async function submit(nextHost = host, nextSlug = nextHost, nextName = nextHost) {
    setConnecting(true);
    setFeedback('');
    try {
      await connect<true>({ body: cleanConnection({ host: nextHost }), throwOnError: true });
      onConnected(nextSlug, nextName);
    } catch (error) {
      const message = errorMessage(error);
      setFeedback(/401|unauthorized/i.test(message)
        ? 'Elasticsearch requires authentication. Configure credentials for this host in the Cerebro backend config.'
        : message);
    } finally {
      setConnecting(false);
    }
  }

  const host = connectForm.getFieldValue('host');

  return (
    <>
      <div className="flex flex-col items-center pb-[60px] pt-20 text-center">
        <CerebroLogo size="login" />
        <div className="text-center">
          <h4>
            Cerebro <small>v{APP_VERSION}</small>
          </h4>
        </div>
      </div>
      <div style={{ maxWidth: 400, marginLeft: 'auto', marginRight: 'auto' }}>
        {authStatus?.enabled && authStatus.authenticated ? (
          <div className="mb-3 flex items-center justify-end gap-3 text-muted">
            {authStatus.user ? <span>{authStatus.user}</span> : null}
            <form action="/auth/logout" method="POST">
              <Button icon="log-out" size="xs" type="submit">
                Logout
              </Button>
            </form>
          </div>
        ) : null}
        <div className="text-center">
          <p>
            {connecting ? (
              <span>
                <Icon name="spinner" spin /> Connecting...
              </span>
            ) : null}
            &nbsp;
            {feedback ? <span className="text-danger">{feedback}</span> : null}
          </p>
        </div>
        {hosts.length > 0 ? (
          <table className="table">
            <thead>
              <tr>
                <th>Known clusters</th>
              </tr>
            </thead>
            <tbody>
              {[...hosts].sort((left, right) => left.name.localeCompare(right.name)).map((knownHost) => (
                <tr key={knownHost.slug}>
                  <td className="normal-action" onClick={() => void submit(knownHost.name, knownHost.slug, knownHost.name)}>
                    <span>{knownHost.name}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : null}
        <form
          onSubmit={(event) => {
            event.preventDefault();
            void connectForm.handleSubmit();
          }}
        >
          <div className="form-group">
            <label htmlFor="host">Node address</label>
            <connectForm.Field name="host">
              {(field) => (
                <input
                  id="host"
                  className="form-control form-control-sm"
                  placeholder="e.g.: http://localhost:9200"
                  type="text"
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                />
              )}
            </connectForm.Field>
          </div>
          <connectForm.Subscribe selector={(state) => state.values.host}>
            {(hostValue) => (
              <Button className="pull-right" disabled={!hostValue} icon="plug" type="submit" variant="success">
                Connect
              </Button>
            )}
          </connectForm.Subscribe>
        </form>
      </div>
    </>
  );
}
