import faviconUrl from '../assets/favicon.png';
import logoUrl from '../assets/logo.png';

type CerebroLogoProps = {
  size: 'header' | 'login';
};

export function CerebroLogo({ size }: CerebroLogoProps) {
  const header = size === 'header';

  return (
    <span className={`relative inline-block ${header ? 'h-[34px] w-[32px]' : ''}`}>
      <img
        alt="Cerebro"
        className={header ? 'block h-[34px] w-auto' : 'block h-40 w-auto'}
        src={header ? faviconUrl : logoUrl}
      />
      <span
        className={
          header
            ? 'absolute -right-[12px] bottom-[-5px] rounded border border-[#00d494]/70 bg-[#2b2f31]/95 px-[4px] py-[1px] text-[11px] font-bold leading-[12px] tracking-[0.08em] text-[#00d494] shadow-[0_0_6px_rgba(0,212,148,0.45)]'
            : 'absolute bottom-1 right-0 rounded border border-[#00d494]/70 bg-[#2b2f31]/95 px-2 py-0.5 text-[13px] font-bold tracking-[0.16em] text-[#00d494] shadow-[0_0_10px_rgba(0,212,148,0.45)]'
        }
      >
        GO
      </span>
    </span>
  );
}
