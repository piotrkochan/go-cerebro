import { useSearch } from '@tanstack/react-router';
import { useEffect, useState } from 'react';

import { loadAuthStatus, type AuthStatus } from '../api/security';
import { CerebroLogo } from '../components/CerebroLogo';
import { Icon } from '../components/Icon';
import { authActions } from '../stores/authStore';
import { sessionActions } from '../stores/sessionStore';
import { APP_VERSION } from '../version';

export function LoginPage() {
  const search = useSearch({ strict: false });
  const invalidLogin = search.error === 'invalid';
  const [authStatus, setAuthStatus] = useState<AuthStatus | null>();
  const loading = authStatus === undefined;
  const passwordLogin = !loading && authStatus?.providers?.password !== false;
  const entraIDLogin = authStatus?.providers?.entraid === true;

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
        {loading ? (
          <div className="text-center">
            <Icon name="spinner" spin /> Loading...
          </div>
        ) : null}
        {passwordLogin ? (
          <form action="/auth/login" className="form-signin" method="POST">
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
        {entraIDLogin ? (
          <div className={passwordLogin ? 'mt-[50px]' : ''}>
            <a className="btn btn-success w-full" href="/auth/entraid/login">
              <Icon name="lock" /> Sign in with Microsoft Entra ID
            </a>
          </div>
        ) : null}
      </div>
    </>
  );
}
