'use client';

import { useEffect, useState } from 'react';
import { PortState, VlanInfo } from '@/lib/types';
import VlanSelect from '@/components/VlanSelect';

interface SetVlanModalProps {
  open: boolean;
  port: PortState | null;
  switchName: string;
  vlans: VlanInfo[];
  applyStatus: 'pending' | 'success' | 'error' | null;
  canEdit: boolean;
  onClose: () => void;
  onApply: (vlan: number) => void;
}

export default function SetVlanModal({
  open,
  port,
  switchName,
  vlans,
  applyStatus,
  canEdit,
  onClose,
  onApply,
}: SetVlanModalProps) {
  const [vlan, setVlan] = useState('');
  const pending = applyStatus === 'pending';

  useEffect(() => {
    if (open) {
      setVlan('');
    }
  }, [open, port?.name]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  if (!open || !port) return null;

  const currentVlan =
    port.accessVlanLabel ?? (port.accessVlan != null ? String(port.accessVlan) : '—');

  return (
    <div className="modal-overlay" onClick={onClose} role="presentation">
      <div
        className="modal-panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="set-vlan-title"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="modal-header">
          <h3 id="set-vlan-title">Set VLAN</h3>
          <button type="button" className="modal-close" onClick={onClose} aria-label="Close">
            <i className="fas fa-times" />
          </button>
        </div>

        <dl className="modal-details">
          <div>
            <dt>Switch</dt>
            <dd>{switchName}</dd>
          </div>
          <div>
            <dt>Port</dt>
            <dd>{port.name}</dd>
          </div>
          {port.description && (
            <div>
              <dt>Description</dt>
              <dd>{port.description}</dd>
            </div>
          )}
          <div>
            <dt>Current VLAN</dt>
            <dd>{currentVlan}</dd>
          </div>
          <div>
            <dt>Link</dt>
            <dd>
              <span className={`pill pill-${port.operState}`}>{port.operState}</span>
            </dd>
          </div>
        </dl>

        <div className="modal-form">
          <label htmlFor="vlan-select">New VLAN</label>
          <VlanSelect
            vlans={vlans}
            value={vlan}
            disabled={!canEdit || pending}
            onChange={setVlan}
            className="modal-vlan-select"
          />
        </div>

        <div className="modal-actions">
          <button type="button" className="btn" onClick={onClose} disabled={pending}>
            Cancel
          </button>
          <button
            type="button"
            className="btn btn-primary"
            disabled={!canEdit || pending || !vlan}
            onClick={() => {
              const n = parseInt(vlan, 10);
              if (n > 0) onApply(n);
            }}
          >
            {pending ? (
              <>
                <i className="fas fa-spinner fa-spin" /> Applying…
              </>
            ) : (
              'Apply VLAN'
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
