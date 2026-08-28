'use client';

import { useEffect, useState } from 'react';
import SetVlanModal from '@/components/SetVlanModal';
import { RemoteHighlight } from '@/hooks/useWebSocket';
import { PortChangeField } from '@/lib/portChanges';
import { SwitchRuntimeState, VlanInfo } from '@/lib/types';

interface SwitchPanelProps {
  sw: SwitchRuntimeState;
  vlans: VlanInfo[];
  onRefresh: (id: string) => void;
  onSetVlan: (switchId: string, port: string, vlan: number) => void;
  getPortApplyStatus: (switchId: string, port: string) => 'pending' | 'success' | 'error' | null;
  getPortRemoteChange?: (switchId: string, port: string) => RemoteHighlight | null;
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
  remoteFields: string[] | undefined,
): string {
  const classes = [operState === 'down' ? 'status-bad' : 'status-ok'];
  if (applyStatus) {
    classes.push(`apply-state-${applyStatus}`);
  } else if (remoteFields && remoteFields.length > 0) {
    classes.push('remote-change');
  }
  return classes.join(' ');
}

function cellClass(base: string, remoteFields: string[] | undefined, field: string): string {
  if (remoteFields?.includes(field)) {
    return `${base} remote-change-field`;
  }
  return base;
}

export default function SwitchPanel({
  sw,
  vlans,
  onRefresh,
  onSetVlan,
  getPortApplyStatus,
  getPortRemoteChange,
  disabled,
}: SwitchPanelProps) {
  const [modalPort, setModalPort] = useState<string | null>(null);
  const panelClass = ['card-panel', sw.online ? 'online' : 'offline'].join(' ');
  const vlanOptions = vlans.length > 0 ? vlans : (sw.vlans ?? []);
  const canEdit = sw.online && !disabled;
  const activePort = modalPort ? sw.ports.find((p) => p.name === modalPort) ?? null : null;
  const modalApplyStatus = modalPort ? getPortApplyStatus(sw.id, modalPort) : null;

  useEffect(() => {
    if (modalPort && modalApplyStatus === 'success') {
      setModalPort(null);
    }
  }, [modalPort, modalApplyStatus]);

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

      <div className="card-body card-body-compact">
        <span className="muted">{sw.ip || 'Not configured'}</span>
        <span className="muted">·</span>
        <span className="muted" title="Last successful poll">
          {formatTime(sw.lastSuccessAt)}
        </span>
      </div>

      <div className="status-table-wrapper">
        <table className="status-table">
          <thead>
            <tr>
              <th>Port</th>
              <th>Description</th>
              <th>Link</th>
              <th>VLAN</th>
              <th className="col-action" aria-label="Actions" />
            </tr>
          </thead>
          <tbody>
            {sw.ports.map((p) => {
              const applyStatus = getPortApplyStatus(sw.id, p.name);
              const remote = getPortRemoteChange?.(sw.id, p.name);
              const remoteFields = remote?.fields as PortChangeField[] | undefined;
              const pending = applyStatus === 'pending';
              return (
                <tr key={p.name} className={rowClass(p.operState, applyStatus, remoteFields)}>
                  <td className="port-name">{p.name}</td>
                  <td
                    className={cellClass('port-description', remoteFields, 'description')}
                    title={p.description || undefined}
                  >
                    {p.description || '—'}
                  </td>
                  <td className={cellClass('', remoteFields, 'link')}>
                    <span className={`pill pill-${p.operState}`}>{p.operState}</span>
                  </td>
                  <td
                    className={cellClass('', remoteFields, 'vlan')}
                    title={p.accessVlanTitle ?? undefined}
                  >
                    {p.accessVlanLabel ?? (p.accessVlan != null ? String(p.accessVlan) : '—')}
                  </td>
                  <td className="col-action">
                    <button
                      type="button"
                      className="btn btn-sm btn-icon"
                      disabled={!canEdit || pending}
                      title="Set VLAN"
                      onClick={() => setModalPort(p.name)}
                    >
                      {pending ? (
                        <i className="fas fa-spinner fa-spin" />
                      ) : (
                        <i className="fas fa-pen" />
                      )}
                    </button>
                  </td>
                </tr>
              );
            })}
            {sw.ports.length === 0 && (
              <tr>
                <td colSpan={5} className="empty-cell">
                  {sw.online ? 'No ports discovered' : 'Offline — no data'}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <SetVlanModal
        open={modalPort !== null}
        port={activePort}
        switchName={sw.name}
        vlans={vlanOptions}
        applyStatus={modalApplyStatus}
        canEdit={canEdit}
        onClose={() => setModalPort(null)}
        onApply={(vlan) => {
          if (modalPort) onSetVlan(sw.id, modalPort, vlan);
        }}
      />

      {sw.lastError && (
        <div className="card-error">
          <i className="fas fa-exclamation-triangle" /> {sw.lastError}
        </div>
      )}
    </div>
  );
}
