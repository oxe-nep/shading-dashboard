'use client';

import ActionFeedbackBanner from '@/components/ActionFeedback';
import SwitchPanel from '@/components/SwitchPanel';
import { useWebSocket } from '@/hooks/useWebSocket';

export default function DashboardPage() {
  const {
    connected,
    snapshot,
    feedback,
    setPortVlan,
    refreshSwitch,
    getPortApplyStatus,
    getPortRemoteChange,
  } = useWebSocket();
  const vlanOptions =
    (snapshot.selectableVlans?.length ?? 0) > 0
      ? snapshot.selectableVlans!
      : (snapshot.discoveredVlans ?? []);

  return (
    <>
      <div className={`connection-status-fixed ${connected ? 'connected' : 'disconnected'}`}>
        <i className="fas fa-circle" />
        <span>{connected ? 'Connected' : 'Disconnected'}</span>
      </div>

      <ActionFeedbackBanner feedback={feedback} />

      {!connected && (
        <div className="loading">
          <div className="loading-spinner">
            <i className="fas fa-spinner fa-spin" />
          </div>
          <span>Connecting to backend…</span>
        </div>
      )}

      <section className="dashboard-section">
        <div className="cards-grid switches-grid">
          {snapshot.switches.map((sw) => (
            <SwitchPanel
              key={sw.id}
              sw={sw}
              vlans={vlanOptions}
              onRefresh={refreshSwitch}
              onSetVlan={setPortVlan}
              getPortApplyStatus={getPortApplyStatus}
              getPortRemoteChange={getPortRemoteChange}
              disabled={!connected}
            />
          ))}
        </div>
      </section>

      {connected && snapshot.switches.length === 0 && (
        <div className="empty-state">
          <i className="fas fa-network-wired" />
          <p>No switches configured. Open settings to add switch credentials.</p>
        </div>
      )}
    </>
  );
}
