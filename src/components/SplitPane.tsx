import { useState, type PointerEvent as ReactPointerEvent, type ReactNode } from 'react';

export function SplitPane({
  left,
  leftMinWidth = 320,
  right,
  rightMinWidth = 320,
  storageKey,
}: {
  left: ReactNode;
  leftMinWidth?: number;
  right: ReactNode;
  rightMinWidth?: number;
  storageKey: string;
}) {
  const [splitPercent, setSplitPercent] = useState(() => savedSplitPercent(storageKey));

  function startResize(event: ReactPointerEvent<HTMLDivElement>) {
    event.preventDefault();
    const container = event.currentTarget.parentElement;
    if (!container) return;
    const rect = container.getBoundingClientRect();
    const pointerId = event.pointerId;
    event.currentTarget.setPointerCapture(pointerId);

    function move(moveEvent: PointerEvent) {
      const next = clampSplit(((moveEvent.clientX - rect.left) / rect.width) * 100);
      setSplitPercent(next);
      window.localStorage.setItem(storageKey, String(Math.round(next)));
    }

    function stop() {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', stop);
      window.removeEventListener('pointercancel', stop);
    }

    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', stop, { once: true });
    window.addEventListener('pointercancel', stop, { once: true });
  }

  return (
    <div className="flex gap-0">
      <div className="pr-[15px]" style={{ flexBasis: `calc(${splitPercent}% - 5px)`, minWidth: leftMinWidth }}>
        {left}
      </div>
      <div
        aria-label="Resize panels"
        className="w-[10px] shrink-0 cursor-col-resize border-x border-[#434749] hover:bg-[#434749]"
        role="separator"
        tabIndex={0}
        onPointerDown={startResize}
      />
      <div className="pl-[15px]" style={{ flexBasis: `calc(${100 - splitPercent}% - 5px)`, minWidth: rightMinWidth }}>
        {right}
      </div>
    </div>
  );
}

function savedSplitPercent(storageKey: string) {
  const value = Number(window.localStorage.getItem(storageKey));
  return clampSplit(Number.isFinite(value) ? value : 50);
}

function clampSplit(value: number) {
  return Math.min(75, Math.max(25, value));
}
