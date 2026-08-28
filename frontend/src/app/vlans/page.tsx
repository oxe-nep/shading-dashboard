'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { fetchConfig, saveConfig } from '@/lib/api';
import { useWebSocket } from '@/hooks/useWebSocket';

export default function VlansPage() {
  const router = useRouter();
  const { connected, snapshot } = useWebSocket();
  const discovered = snapshot.discoveredVlans ?? [];
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [configLoaded, setConfigLoaded] = useState(false);
  const [initialized, setInitialized] = useState(false);
  const savedAllowedRef = useRef<number[]>([]);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    fetchConfig()
      .then((cfg) => {
        savedAllowedRef.current = cfg.allowedVlans ?? [];
        setConfigLoaded(true);
      })
      .catch(console.error);
  }, []);

  useEffect(() => {
    if (!configLoaded || initialized || discovered.length === 0) return;

    const saved = savedAllowedRef.current;
    if (saved.length === 0) {
      setSelected(new Set(discovered.map((v) => v.id)));
    } else {
      setSelected(new Set(saved));
    }
    setInitialized(true);
  }, [configLoaded, discovered, initialized]);

  const rows = useMemo(
    () => [...discovered].sort((a, b) => a.id - b.id),
    [discovered],
  );

  const toggleVlan = (id: number) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const selectAll = () => {
    setSelected(new Set(discovered.map((v) => v.id)));
  };

  const handleSave = async () => {
    if (!configLoaded) return;

    const selectedIds = Array.from(selected).sort((a, b) => a - b);
    if (selectedIds.length === 0) {
      setMessage('Select at least one VLAN for the dropdowns.');
      return;
    }

    const allIds = discovered.map((v) => v.id).sort((a, b) => a - b);
    const allSelected =
      allIds.length > 0 &&
      allIds.length === selectedIds.length &&
      allIds.every((id, i) => id === selectedIds[i]);

    setSaving(true);
    setMessage('');
    try {
      const config = await fetchConfig();
      await saveConfig({
        ...config,
        allowedVlans: allSelected ? [] : selectedIds,
      });
      router.push('/');
    } catch (err) {
      setMessage(String(err));
      setSaving(false);
    }
  };

  if (!configLoaded) {
    return (
      <div className="loading">
        <div className="loading-spinner">
          <i className="fas fa-spinner fa-spin" />
        </div>
        <span>Loading VLAN settings…</span>
      </div>
    );
  }

  return (
    <div className="config-page vlans-page">
      <div className="config-section">
        <h2>VLAN selection</h2>
        <p className="config-intro">
          All VLANs discovered on your switches are listed below. Uncheck any you do not want in the
          port assignment dropdowns.
        </p>

        <div className="vlan-toolbar">
          <button type="button" className="btn btn-sm" onClick={selectAll} disabled={discovered.length === 0}>
            Select all
          </button>
          <span className="vlan-count">
            {selected.size} of {discovered.length} included
          </span>
        </div>

        <div className="config-table-wrapper">
          <table className="config-table vlan-table">
            <thead>
              <tr>
                <th className="vlan-check-col">In dropdown</th>
                <th>VLAN ID</th>
                <th>Name</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((v) => (
                <tr key={v.id} className={selected.has(v.id) ? '' : 'vlan-row-excluded'}>
                  <td className="vlan-check-col">
                    <input
                      type="checkbox"
                      checked={selected.has(v.id)}
                      onChange={() => toggleVlan(v.id)}
                    />
                  </td>
                  <td>{v.id}</td>
                  <td title={v.title}>{v.name || v.label}</td>
                </tr>
              ))}
              {rows.length === 0 && (
                <tr>
                  <td colSpan={3} className="empty-cell">
                    {connected
                      ? 'No VLANs discovered yet — check switch connectivity'
                      : 'Waiting for backend connection…'}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {message && <p className="config-message error">{message}</p>}
        <div className="config-actions">
          <button
            type="button"
            className="btn btn-primary"
            disabled={saving || discovered.length === 0}
            onClick={() => void handleSave()}
          >
            {saving ? 'Saving…' : 'Save and return'}
          </button>
        </div>
      </div>
    </div>
  );
}
