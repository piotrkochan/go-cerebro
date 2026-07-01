import type { ReactNode } from 'react';

import { Checkbox } from './Checkbox';
import { Icon } from './Icon';

export type SettingsPageGroup<TSetting> = {
  name: string;
  settings: TSetting[];
};

type SettingsPageLayoutProps<TSetting> = {
  actions?: ReactNode;
  filterName: string;
  groupAriaLabel: string;
  groupPrefix: string;
  groups: SettingsPageGroup<TSetting>[];
  label: string;
  pendingContent: ReactNode;
  pendingCount: number;
  pendingFooter?: ReactNode;
  renderSetting: (setting: TSetting) => ReactNode;
  showStatic: boolean;
  title: ReactNode;
  onFilterNameChange: (value: string) => void;
  onShowStaticChange: (value: boolean) => void;
};

export function SettingsPageLayout<TSetting>({
  actions,
  filterName,
  groupAriaLabel,
  groupPrefix,
  groups,
  label,
  pendingContent,
  pendingCount,
  pendingFooter,
  renderSetting,
  showStatic,
  title,
  onFilterNameChange,
  onShowStaticChange,
}: SettingsPageLayoutProps<TSetting>) {
  const visibleSettings = groups.reduce((total, group) => total + group.settings.length, 0);

  return (
    <div className="grid gap-[15px] lg:grid-cols-[180px_minmax(0,1fr)_260px] 2xl:grid-cols-[220px_minmax(0,1fr)_320px]">
        <aside className="lg:sticky lg:top-[70px] lg:self-start">
          <div className="border border-[#55595c] bg-[#373a3c]">
            <div className="border-b border-[#55595c] px-3 py-2">
              <div className="text-[12px] uppercase text-[#8b8f95]">groups</div>
              <div className="text-[18px] font-light">{visibleSettings} settings</div>
            </div>
            <nav className="settings-group-nav max-h-[calc(100vh-150px)] overflow-y-auto p-1" aria-label={groupAriaLabel}>
              {groups.map((group) => (
                <button
                  className="flex w-full cursor-pointer items-center justify-between px-2 py-1.5 text-left text-[#d0d0d0] hover:bg-[#434749] hover:text-white"
                  key={group.name}
                  type="button"
                  onClick={() => scrollToGroup(groupID(groupPrefix, group.name))}
                >
                  <span className="truncate">{group.name}</span>
                  <span className="ml-2 text-[#8b8f95]">{group.settings.length}</span>
                </button>
              ))}
            </nav>
          </div>
        </aside>

        <main>
          <div className="sticky top-[55px] z-20 -mt-[15px] mb-[15px] bg-[#373a3c] pt-[15px]">
            <div className="border border-[#55595c] bg-[#373a3c]">
              <div className="flex flex-col gap-2 border-b border-[#55595c] px-3 py-3 sm:flex-row sm:items-center sm:justify-between">
                <div className="min-w-0">
                  <div className="text-[12px] uppercase text-[#8b8f95]">{label}</div>
                  <h1 className="m-0 break-all text-[18px] font-normal">{title}</h1>
                </div>
                {actions ? <div className="shrink-0">{actions}</div> : null}
              </div>
              <div className="flex flex-col gap-3 p-3 lg:flex-row lg:items-center">
                <div className="min-w-0 flex-1">
                  <input className="form-control" placeholder="filter settings by name" value={filterName} onChange={(event) => onFilterNameChange(event.target.value)} />
                </div>
                <Checkbox
                  checked={showStatic}
                  className="whitespace-nowrap text-[#d0d0d0]"
                  label={<>show static settings <Icon className="alert-warning" name="lock" /></>}
                  onChange={onShowStaticChange}
                />
              </div>
            </div>
          </div>

          <div className="space-y-[15px]">
            {groups.map((group) => (
              <section className="scroll-mt-[215px] border border-[#55595c]" id={groupID(groupPrefix, group.name)} key={group.name}>
                <header className="flex items-center justify-between border-b border-[#55595c] bg-[#34383a] px-3 py-2">
                  <h2 className="m-0 text-[13px] font-bold uppercase">{group.name}</h2>
                  <span className="text-[#8b8f95]">{group.settings.length}</span>
                </header>
                <div className="divide-y divide-[#55595c]">{group.settings.map(renderSetting)}</div>
              </section>
            ))}
          </div>
        </main>

        <aside className="lg:sticky lg:top-[70px] lg:self-start">
          <div className="border border-[#55595c] bg-[#373a3c]">
            <div className="border-b border-[#55595c] px-3 py-2">
              <h2 className="m-0 text-[14px] font-normal">
                pending changes <small className="info-text">({pendingCount})</small>
              </h2>
            </div>
            {pendingCount ? (
              <>
                <div className="max-h-[calc(100vh-180px)] divide-y divide-[#55595c] overflow-y-auto">{pendingContent}</div>
                {pendingFooter ? <div className="border-t border-[#55595c] p-3 text-right">{pendingFooter}</div> : null}
              </>
            ) : (
              <div className="p-3 text-[#8b8f95]">no pending changes</div>
            )}
          </div>
        </aside>
      </div>
  );
}

function groupID(prefix: string, name: string) {
  return `${prefix}-${name.replace(/[^a-z0-9_-]+/gi, '-')}`;
}

function scrollToGroup(id: string) {
  document.getElementById(id)?.scrollIntoView({ block: 'start', behavior: 'smooth' });
}
