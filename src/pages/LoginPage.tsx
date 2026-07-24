import { useSearch } from '@tanstack/react-router';
import { useEffect, useState } from 'react';

import { apiURL, loadAuthStatus, type AuthStatus } from '../api/security';
import { CerebroLogo } from '../components/CerebroLogo';
import { Icon } from '../components/Icon';
import { authActions } from '../stores/authStore';
import { sessionActions } from '../stores/sessionStore';
import { APP_VERSION } from '../version';

export function LoginPage() {
  const search = useSearch({ strict: false });
  const invalidLogin = search.error === 'invalid';
  const externalLoginFailed = search.error === 'external';
  const [authStatus, setAuthStatus] = useState<AuthStatus | null>();
  const loading = authStatus === undefined;
  const passwordLogin = !loading && authStatus?.providers?.password !== false;
  const externalProviders = authStatus?.external_providers?.length ? authStatus.external_providers : legacyExternalProviders(authStatus);

  useEffect(() => {
    let cancelled = false;
    authActions.clear();
    sessionActions.disconnect();
    loadAuthStatus().then((status) => {
      if (!cancelled) setAuthStatus(status);
    });
    return () => {
      cancelled = true;
    };
  }, []);

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
      <div className="mx-auto max-w-[300px]">
        {invalidLogin ? (
          <div className="alert alert-danger" role="alert">
            Invalid username or password.
          </div>
        ) : null}
        {externalLoginFailed ? (
          <div className="alert alert-danger" role="alert">
            External sign-in failed. Check the server logs and identity provider configuration.
          </div>
        ) : null}
        {loading ? (
          <div className="text-center">
            <Icon name="spinner" spin /> Loading...
          </div>
        ) : null}
        {passwordLogin ? (
          <form action={apiURL('/auth/login')} className="form-signin mb-[14px] flow-root" method="POST">
            <div className="form-group">
              <label className="sr-only" htmlFor="inputUser">User</label>
              <input autoFocus required className="form-control form-control-sm" id="inputUser" name="user" placeholder="User" type="text" />
            </div>
            <div className="form-group">
              <label className="sr-only" htmlFor="inputPassword">Password</label>
              <input required className="form-control form-control-sm" id="inputPassword" name="password" placeholder="Password" type="password" />
            </div>
            <button className="btn btn-success pull-right" type="submit">
              <Icon name="plug" /> Sign in
            </button>
          </form>
        ) : null}
        <div className="space-y-[10px]">
          {externalProviders.map((provider) => (
            <a className="btn btn-success w-full" href={apiURL(provider.login_path || '/')} key={`${provider.kind}-${provider.id}`}>
              <Icon name="lock" /> Sign in with {provider.name || provider.kind || 'provider'}
            </a>
          ))}
        </div>
      </div>
    </>
  );
}

function legacyExternalProviders(status: AuthStatus | null | undefined) {
  const providers = [];
  if (status?.providers?.entraid === true) {
    providers.push({ id: 'default', kind: 'entra_id', login_path: '/auth/entraid/login', name: 'Microsoft Entra ID' });
  }
  if (status?.providers?.oauth === true) {
    providers.push({
      id: 'default',
      kind: 'oauth',
      login_path: '/auth/oauth/login',
      name: status.provider_names?.oauth || status.providers?.oauth_name || 'OAuth',
    });
  }
  return providers;
}
