import { useEffect, useState } from 'react';
import { useForm } from '@tanstack/react-form';

import { connect, connectHosts, type HostBodyWritable } from '../api/client';
import { Icon } from '../components/Icon';
import { authFormDefaults, connectFormDefaults } from '../forms/connectForm';
import type { ConnectionAuth } from '../types';
import { cleanConnection } from '../utils/connection';
import { errorMessage } from '../utils/format';

export function ConnectPage({
  currentHost,
  onConnected,
}: {
  currentHost: string;
  onConnected: (host: string, auth?: ConnectionAuth) => void;
}) {
  const [hosts, setHosts] = useState<string[]>([]);
  const [connecting, setConnecting] = useState(false);
  const [feedback, setFeedback] = useState('');
  const [unauthorized, setUnauthorized] = useState(false);
  const connectForm = useForm({
    defaultValues: connectFormDefaults(currentHost),
    onSubmit: async ({ value }) => {
      await submit(value.host);
    },
  });
  const authForm = useForm({
    defaultValues: authFormDefaults,
    onSubmit: async ({ value }) => {
      await submit(connectForm.getFieldValue('host'), value);
    },
  });

  useEffect(() => {
    let ignore = false;
    async function load() {
      try {
        const result = await connectHosts<true>({ throwOnError: true });
        if (!ignore) setHosts(result.data.items ?? []);
      } catch {
        if (!ignore) setHosts([]);
      }
    }
    void load();
    return () => {
      ignore = true;
    };
  }, []);

  useEffect(() => {
    connectForm.setFieldValue('host', currentHost);
  }, [connectForm, currentHost]);

  async function submit(nextHost = host, nextAuth: Pick<HostBodyWritable, 'username' | 'password'> = {}) {
    setConnecting(true);
    setFeedback('');
    try {
      await connect<true>({ body: cleanConnection({ host: nextHost, ...nextAuth }), throwOnError: true });
      onConnected(nextHost, nextAuth);
    } catch (error) {
      const message = errorMessage(error);
      setFeedback(message);
      setUnauthorized(/401|unauthorized/i.test(message));
    } finally {
      setConnecting(false);
    }
  }

  const host = connectForm.getFieldValue('host');

  return (
    <>
      <div className="row" style={{ paddingTop: 80, paddingBottom: 60 }}>
        <div className="col-xs-12 text-center">
          <img src="/img/logo.png" height="160" />
          <h4>
            Cerebro <small>v0.9.4</small>
          </h4>
        </div>
      </div>
      <div style={{ maxWidth: 400, marginLeft: 'auto', marginRight: 'auto' }}>
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
        {!unauthorized ? (
          <>
            {hosts.length > 0 ? (
              <table className="table">
                <thead>
                  <tr>
                    <th>Known clusters</th>
                  </tr>
                </thead>
                <tbody>
                  {[...hosts].sort().map((knownHost) => (
                    <tr key={knownHost}>
                      <td className="normal-action" onClick={() => void submit(knownHost)}>
                        <span>{knownHost}</span>
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
                  <button className="btn btn-success pull-right" disabled={!hostValue} type="submit">
                    Connect
                  </button>
                )}
              </connectForm.Subscribe>
            </form>
          </>
        ) : (
          <form
            onSubmit={(event) => {
              event.preventDefault();
              void authForm.handleSubmit();
            }}
          >
            <div className="form-group">
              <label htmlFor="username">Username</label>
              <authForm.Field name="username">
                {(field) => (
                  <input
                    id="username"
                    className="form-control form-control-sm"
                    placeholder="admin"
                    type="text"
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                )}
              </authForm.Field>
            </div>
            <div className="form-group">
              <label htmlFor="password">Password</label>
              <authForm.Field name="password">
                {(field) => (
                  <input
                    id="password"
                    className="form-control form-control-sm"
                    type="password"
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                )}
              </authForm.Field>
            </div>
            <button className="btn btn-success pull-right" type="submit">
              Authenticate
            </button>
          </form>
        )}
      </div>
    </>
  );
}
