import { ReactNode } from 'react';

interface OverlayModalProps {
  open: boolean;
  onClose: () => void;
  children: ReactNode;
}

export function OverlayModal({ open, onClose, children }: OverlayModalProps) {
  if (!open) return null;
  return (
    <div className="modal is-open" aria-hidden={!open} style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, zIndex: 999999 }}>
      <div className="modal-overlay" style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, background: 'rgba(40,50,50,0.6)', zIndex: 999999 }} onClick={onClose} />
      <div className="modal-content" style={{ position: 'fixed', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', zIndex: 1000000, background: '#fff', borderRadius: 8, boxShadow: '0 6px 40px rgba(0,0,0,0.18)', maxWidth: 600, width: '90vw', padding: 32 }}>
        <button onClick={onClose} style={{ position: 'absolute', top: 8, right: 12, background: 'none', border: 'none', fontSize: 26, color: '#888', cursor: 'pointer' }} aria-label="Close">×</button>
        {children}
      </div>
    </div>
  );
}