import { Link } from '@tanstack/react-router';

import { Icon } from './Icon';
import { timeInterval } from '../utils/format';

const refreshOptions = [5000, 10000, 15000, 30000, 60000];

export function Navbar({
  connected,
  disconnect,
  host,
  refreshInterval,
  setRefreshInterval,
  status,
}: {
  connected: boolean;
  disconnect: () => void;
  host: string;
  refreshInterval: number;
  setRefreshInterval: (value: number) => void;
  status: string;
}) {
  if (!connected) return null;
  const search = { host };
  const statusBorder =
    status === 'green'
      ? 'border-[#1AC98E]'
      : status === 'yellow'
        ? 'border-[#E4D836]'
        : status === 'red'
          ? 'border-[#E64759]'
          : 'border-[#55595c]';
  const navLink = 'flex h-[50px] items-center gap-1.5 px-[15px] text-[#eceeef] hover:bg-[#434749] hover:text-white';
  const menu =
    'absolute top-full left-0 z-[1000] hidden min-w-[160px] list-none border border-[#55595c] bg-[#373a3c] py-[5px] text-left shadow-lg group-hover:block group-focus-within:block [&>li>a]:flex [&>li>a]:items-center [&>li>a]:gap-1.5 [&>li>a]:whitespace-nowrap [&>li>a]:px-5 [&>li>a]:py-[3px] [&>li>a:hover]:bg-[#434749] [&>li>a:hover]:text-white';

  return (
    <nav className={`fixed top-0 right-0 left-0 z-[1030] min-h-[50px] border-b-[5px] bg-[#373a3c] ${statusBorder}`}>
      <div className="mx-auto flex min-h-[50px] w-full items-stretch px-[15px]">
        <div className="flex items-stretch">
          <button className="hidden" type="button">
            <span className="sr-only">Toggle navigation</span>
            <span className="icon-bar" />
            <span className="icon-bar" />
            <span className="icon-bar" />
          </button>
          <Link className="flex h-[50px] items-start px-[15px]" search={search} to="/overview">
            <img src="/img/favicon.png" width="28" style={{ paddingTop: 6 }} />
          </Link>
        </div>
        <div id="navbar" className="flex flex-1 items-stretch justify-between">
          <ul className="m-0 flex list-none p-0">
            <li>
              <Link className={navLink} search={search} to="/overview">
                <Icon name="sitemap" /> overview
              </Link>
            </li>
            <li>
              <Link className={navLink} search={search} to="/nodes">
                <Icon name="server" /> nodes
              </Link>
            </li>
            <li>
              <Link className={navLink} search={search} to="/rest">
                <Icon name="edit" /> rest
              </Link>
            </li>
            <li className="group relative">
              <a className={navLink} href="#more">
                <Icon name="magic" /> more <Icon name="caret-down" />
              </a>
              <ul className={menu}>
                <li>
                  <Link search={search} to="/create">
                    <Icon name="file" /> create index
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/cluster_settings">
                    <Icon name="cogs" /> cluster settings
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/aliases">
                    <Icon name="tags" /> aliases
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/analysis">
                    <Icon name="puzzle" /> analysis
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/templates">
                    <Icon name="book" /> index templates
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/repositories">
                    <Icon name="database" /> repositories
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/snapshot">
                    <Icon name="camera" /> snapshot
                  </Link>
                </li>
                <li>
                  <Link search={search} to="/cat">
                    <Icon name="list" /> cat apis
                  </Link>
                </li>
              </ul>
            </li>
          </ul>
          <ul className="m-0 ml-auto flex list-none p-0">
            <li className="group relative">
              <a className={navLink} href="#refresh">
                <Icon name="refresh" /> {timeInterval(refreshInterval)} <Icon name="caret-down" />
              </a>
              <ul className={`${menu} min-w-[60px]`}>
                {refreshOptions.map((value) => (
                  <li key={value}>
                    <a className="cursor-pointer" onClick={() => setRefreshInterval(value)}>
                      {timeInterval(value)}
                    </a>
                  </li>
                ))}
              </ul>
            </li>
            <li className="hidden sm:block">
              <a className={navLink}>
                {host} [{status}]
              </a>
            </li>
            <li>
              <a className={`${navLink} hidden cursor-pointer sm:flex`} onClick={disconnect}>
                <Icon name="plug" />
              </a>
            </li>
          </ul>
        </div>
      </div>
    </nav>
  );
}
