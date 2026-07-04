import { useEffect, useState } from 'react';
import { useStore } from '@tanstack/react-store';

import { loadAuthMe } from '../api/security';
import { DataTable, type DataTableColumn } from '../components/DataTable';
import { Icon } from '../components/Icon';
import { authStore, type AuthPermission } from '../stores/authStore';
import { errorMessage } from '../utils/format';

export function ProfilePage() {
  const auth = useStore(authStore);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    let ignore = false;
    async function load() {
      setLoading(true);
      setError('');
      try {
        await loadAuthMe();
      } catch (err) {
        if (!ignore) setError(errorMessage(err));
      } finally {
        if (!ignore) setLoading(false);
      }
    }
    void load();
    return () => {
      ignore = true;
    };
  }, []);

  const permissionColumns: DataTableColumn<AuthPermission>[] = [
    { header: 'resource', key: 'resource', render: (row) => row.resource || <span className="info-text">-</span> },
    { header: 'action', key: 'action', render: (row) => row.action || <span className="info-text">-</span> },
    { header: 'object', key: 'object', render: (row) => row.object || <span className="info-text">-</span> },
    { header: 'effect', key: 'effect', render: (row) => <span className={row.effect === 'deny' ? 'text-[#E64759]' : 'text-[#1AC98E]'}>{row.effect || '-'}</span> },
  ];

  return (
    <div className="row">
      <div className="col-sm-12">
        <h4 className="mb-[15px] flex items-center gap-2">
          <Icon name="user" /> profile
          {loading ? <span className="info-text">loading...</span> : null}
        </h4>
        {error ? <div className="alert alert-danger">{error}</div> : null}
      </div>

      <div className="col-sm-4">
        <table className="table table-condensed">
          <tbody>
            <ProfileRow label="user" value={auth.user || '-'} />
            <ProfileRow label="provider" value={auth.provider || '-'} />
            <ProfileRow label="authenticated" value={auth.authenticated ? 'yes' : 'no'} />
          </tbody>
        </table>
      </div>

      <div className="col-sm-4">
        <h4>roles <small className="info-text">({auth.roles.length})</small></h4>
        <ValueList values={auth.roles} />
      </div>

      <div className="col-sm-4">
        <h4>groups <small className="info-text">({auth.groups.length})</small></h4>
        <ValueList values={auth.groups} />
      </div>

      <div className="col-sm-12 mt-[20px]">
        <h4>permissions <small className="info-text">({auth.permissions.length})</small></h4>
        <DataTable
          columns={permissionColumns}
          empty="no permissions assigned"
          getRowKey={(permission, index) => `${permission.resource}-${permission.action}-${permission.object}-${permission.effect}-${index}`}
          rows={auth.permissions}
        />
      </div>
    </div>
  );
}

function ProfileRow({ label, value }: { label: string; value: string }) {
  return (
    <tr>
      <td className="info-text w-[130px]">{label}</td>
      <td>{value}</td>
    </tr>
  );
}

function ValueList({ values }: { values: string[] }) {
  if (!values.length) return <div className="info-text">none</div>;
  return (
    <ul className="m-0 list-none p-0">
      {values.map((value) => (
        <li className="border-b border-[#55595c] py-[4px]" key={value}>
          {value}
        </li>
      ))}
    </ul>
  );
}
