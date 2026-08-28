'use client';

import { useEffect, useState } from 'react';
import { SwitchRuntimeState, VlanInfo } from '@/lib/types';
import VlanSelect from '@/components/VlanSelect';

interface SwitchPanelProps {
  sw: SwitchRuntimeState;
  vlans: VlanInfo[];
  onRefresh: (id: string) => void;
  onSetVlan: (switchId: string, port: string, vlan: number) => void;
  getPortApplyStatus: (switchId: string, port: string) => 'pending' | 'success' | 'error' | null;
  disabled?: boolean;
}

function formatTime(iso: string | null): string {
  if (!iso) return '—';
  try {
    return new Date(iso).toLocaleTimeString();
  } catch {
    return iso;
  }
}

function rowClass(
  operState: string,
  applyStatus: 'pending' | 'success' | 'error' | null,
): string {
  const classes = [operState === 'down' ? 'status-bad' : 'status-ok'];
  if (applyStatus) {
    classes.push(`apply-state-${applyStatus}`);
  }
  return classes.join(' ');
}

export default function SwitchPanel({
  sw,
  vlans,
  onRefresh,
  onSetVlan,
  getPortApplyStatus,
  disabled,
}: SwitchPanelProps) {
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const panelClass = ['card-panel', sw.online ? 'online' : 'offline'].join(' ');
  const vlanOptions = vlans.length > 0 ? vlans : (sw.vlans ?? []);
  const canEdit = sw.online && !disabled;

  useEffect(() => {
    setDrafts((current) => {
      const next = { ...current };
      let changed = false;
      for (const portName of Object.keys(current)) {
        if (getPortApplyStatus(sw.id, portName) === 'success') {
          delete next[portName];
          changed = true;
        }
      }
      return changed ? next : current;
    });
  }, [getPortApplyStatus, sw.id]);

  return (
    <div className={panelClass}>
      <div className="card-header">
        <div className="card-title">
          <span className={`status-dot ${sw.online ? 'online' : 'offline'}`} />
          <h3>{sw.name}</h3>
        </div>
        <button
          type="button"
          className="card-refresh-btn"
          onClick={() => onRefresh(sw.id)}
          disabled={sw.polling || disabled}
          title="Refresh this switch"
        >
          <i className={`fas fa-sync ${sw.polling ? 'fa-spin' : ''}`} />
        </button>
      </div>

      <div className="card-body">
        <div className="info-row">
          <span className="label">Switch IP</span>
          <span className="value">{sw.ip || 'Not configured'}</span>
        </div>
        <div className="info-row">
          <span className="label">Last successful poll</span>
          <span className="value">{formatTime(sw.lastSuccessAt)}</span>
        </div>
      </div>

      <div className="status-table-wrapper">
        <table className="status-table">
          <thead>
            <tr>
              <th>Port</th>
              <th>Link</th>
              <th>VLAN</th>
              <th>Set VLAN</th>
            </tr>
          </thead>
          <tbody>
            {sw.ports.map((p) => {
              const applyStatus = getPortApplyStatus(sw.id, p.name);
              const pending = applyStatus === 'pending';
              return (
                <tr key={p.name} className={rowClass(p.operState, applyStatus)}>
                  <td>{p.name}</td>
                  <td>
                    <span className={`pill pill-${p.operState}`}>{p.operState}</span>
                  </td>
                  <td title={p.accessVlanTitle ?? undefined}>
                    {p.accessVlanLabel ?? (p.accessVlan != null ? String(p.accessVlan) : '—')}
                  </td>
                  <td>
                    <div className="vlan-set-row">
                      <VlanSelect
                        vlans={vlanOptions}
                        value={drafts[p.name] ?? ''}
                        disabled={!canEdit || pending}
                        onChange={(value) =>
                          setDrafts((d) => ({ ...d, [p.name]: value }))
                        }
                      />
                      <button
                        type="button"
                        className="btn btn-sm btn-primary"
                        disabled={!canEdit || pending || !drafts[p.name]}
                        onClick={() => {
                          const vlan = parseInt(drafts[p.name], 10);
                          if (vlan > 0) onSetVlan(sw.id, p.name, vlan);
                        }}
                      >
                        {pending ? (
                          <>
                            <i className="fas fa-spinner fa-spin" /> …
                          </>
                        ) : (
                          'Apply'
                        )}
                      </button>
                    </div>
                  </td>
                </tr>
              );
            })}
            {sw.ports.length === 0 && (
              <tr>
                <td colSpan={4} className="empty-cell">
                  {sw.online ? 'No ports discovered' : 'Offline — no data'}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {sw.lastError && (
        <div className="card-error">
          <i className="fas fa-exclamation-triangle" /> {sw.lastError}
        </div>
      )}
    </div>
  );
}
