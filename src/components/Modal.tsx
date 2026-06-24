import { useEffect, type ReactNode } from 'react';

export function ModalFrame({
  children,
  dialogClassName = 'modal-lg',
  onClose,
  title,
}: {
  children: ReactNode;
  dialogClassName?: string;
  onClose: () => void;
  title: string;
}) {
  return (
    <>
      <div className="modal-backdrop in" onClick={onClose} />
      <div className="modal in !block" tabIndex={-1}>
        <div className={`modal-dialog ${dialogClassName}`}>
          <div className="modal-content">
            <div className="modal-header flex items-center justify-between">
              <h4 className="modal-title !m-0">{title}</h4>
              <button
                aria-label="close"
                className="close !float-none !text-[#eceeef] !opacity-[.85] [text-shadow:none] hover:!text-white hover:!opacity-100 focus:!text-white focus:!opacity-100"
                type="button"
                onClick={onClose}
              >
                &times;
              </button>
            </div>
            {children}
          </div>
        </div>
      </div>
    </>
  );
}

export function ConfirmModal({
  body,
  confirmClassName = 'btn-danger',
  confirmLabel,
  onClose,
  onConfirm,
  title,
}: {
  body: ReactNode;
  confirmClassName?: string;
  confirmLabel: ReactNode;
  onClose: () => void;
  onConfirm: () => Promise<void> | void;
  title: string;
}) {
  useEscape(onClose);

  async function confirm() {
    onClose();
    await onConfirm();
  }

  return (
    <ModalFrame dialogClassName="" onClose={onClose} title={title}>
      <div className="modal-body">{body}</div>
      <div className="modal-footer">
        <button className="btn btn-default" type="button" onClick={onClose}>
          cancel
        </button>
        <button className={`btn ${confirmClassName}`} type="button" onClick={() => void confirm()}>
          {confirmLabel}
        </button>
      </div>
    </ModalFrame>
  );
}

export function useEscape(onEscape: () => void) {
  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') onEscape();
    }

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [onEscape]);
}
