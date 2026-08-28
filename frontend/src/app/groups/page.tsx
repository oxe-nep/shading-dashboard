'use client';

import { useEffect, useMemo, useState } from 'react';
import ActionFeedbackBanner from '@/components/ActionFeedback';
import GroupVlanApply from '@/components/GroupVlanApply';
import { useWebSocket } from '@/hooks/useWebSocket';
import { fetchConfig, savePortGroups } from '@/lib/api';
import { AppConfig, PortGroupConfig, PortGroupMember, PortGroupRuntime } from '@/lib/types';

function emptyGroup(index: number): PortGroupConfig {
  return {
    id: `grp-${Date.now()}-${index}`,
    name: `Workstation ${index}`,
    members: [],
  };
}

function runtimeForGroup(
  group: PortGroupConfig,
  runtimeGroups: PortGroupRuntime[],
): PortGroupRuntime {
  return (
    runtimeGroups.find((g) => g.id === group.id) ?? {
      id: group.id,
      name: group.name,
      members: group.members,
      currentVlan: null,
      vlanLabel: '—',
      mixed: false,
    }
  );
}

export default function GroupsPage() {
  const {
    connected,
    snapshot,
    feedback,
    setGroupVlan,
    getGroupApplyStatus,
  } = useWebSocket();
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [savedSnapshot, setSavedSnapshot] = useState<PortGroupConfig[]>([]);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');

  const vlanOptions =
    (snapshot.selectableVlans?.length ?? 0) > 0
      ? snapshot.selectableVlans!
      : (snapshot.discoveredVlans ?? []);

  useEffect(() => {
    fetchConfig()
      .then((cfg) => {
        setConfig(cfg);
        setSavedSnapshot(cfg.portGroups);
      })
      .catch(console.error);
  }, []);

  const updateGroup = (index: number, patch: Partial<PortGroupConfig>) => {
    if (!config) return;
    const portGroups = [...config.portGroups];
    portGroups[index] = { ...portGroups[index], ...patch };
    setConfig({ ...config, portGroups });
  };

  const addGroup = () => {
    if (!config) return;
    setConfig({
      ...config,
      portGroups: [...config.portGroups, emptyGroup(config.portGroups.length + 1)],
    });
  };

  const removeGroup = (index: number) => {
    if (!config) return;
    setConfig({
      ...config,
      portGroups: config.portGroups.filter((_, i) => i !== index),
    });
  };

  const toggleMember = (groupIndex: number, switchId: string, port: string) => {
    if (!config) return;
    const group = config.portGroups[groupIndex];
    const key = `${switchId}|${port}`;
    const exists = group.members.some((m) => `${m.switchId}|${m.port}` === key);
    let members: PortGroupMember[];
    if (exists) {
      members = group.members.filter((m) => `${m.switchId}|${m.port}` !== key);
    } else {
      members = [...group.members, { switchId, port }];
    }
    updateGroup(groupIndex, { members });
  };

  const handleSave = async () => {
    if (!config) return;
    if (config.portGroups.length === 0) {
      setMessage('Add at least one port group before saving.');
      return;
    }
    if (config.portGroups.some((g) => !g.name.trim())) {
      setMessage('Each group must have a name.');
      return;
    }
    if (config.portGroups.some((g) => g.members.length === 0)) {
      setMessage('Each group must include at least one port.');
      return;
    }
    setSaving(true);
    setMessage('');
    try {
      const saved = await savePortGroups(config.portGroups);
      setConfig(saved);
      setSavedSnapshot(saved.portGroups);
      setMessage('Groups saved.');
      setSaving(false);
    } catch (err) {
      setMessage(String(err));
      setSaving(false);
    }
  };

  const dirty = useMemo(() => {
    if (!config) return false;
    return JSON.stringify(config.portGroups) !== JSON.stringify(savedSnapshot);
  }, [config, savedSnapshot]);

  const isGroupSaved = (group: PortGroupConfig) => {
    const saved = savedSnapshot.find((g) => g.id === group.id);
    if (!saved) return false;
    return JSON.stringify(saved) === JSON.stringify(group);
  };

  if (!config) {
    return (
      <div className="loading">
        <div className="loading-spinner">
          <i className="fas fa-spinner fa-spin" />
        </div>
        <span>Loading…</span>
      </div>
    );
  }

  return (
    <div className="config-page">
      <ActionFeedbackBanner feedback={feedback} />

      <div className={`connection-status-fixed ${connected ? 'connected' : 'disconnected'}`}>
        <i className="fas fa-circle" />
        <span>{connected ? 'Connected' : 'Disconnected'}</span>
      </div>

      <div className="config-section">
        <h2>Port groups</h2>
        <p className="config-intro">
          Group ports by workstation and apply a VLAN to all members at once.
        </p>

        {config.portGroups.length === 0 && (
          <p className="vlan-hint">No port groups yet. Click &quot;Add group&quot; to create one.</p>
        )}

        {config.portGroups.map((group, gi) => {
          const runtime = runtimeForGroup(group, snapshot.portGroups);
          const isSaved = isGroupSaved(group);
          return (
            <div key={group.id} className="group-editor card-panel">
              <div className="group-editor-header">
                <input
                  type="text"
                  className="group-name-input"
                  value={group.name}
                  placeholder="Group name"
                  onChange={(e) => updateGroup(gi, { name: e.target.value })}
                />
                <button type="button" className="btn btn-sm btn-danger" onClick={() => removeGroup(gi)}>
                  Remove
                </button>
              </div>

              <GroupVlanApply
                group={runtime}
                vlans={vlanOptions}
                applyStatus={getGroupApplyStatus(group.id)}
                onApplyVlan={setGroupVlan}
                disabled={!connected}
                saved={isSaved}
              />

              <div className="port-picker">
                {config.switches
                  .filter((sw) => sw.ip.trim() !== '')
                  .map((sw) => (
                    <div key={sw.id} className="port-picker-switch">
                      <h4>{sw.name}</h4>
                      <p className="muted picker-hint">Ports Gi1/0/1–48</p>
                      <div className="port-checkboxes">
                        {Array.from({ length: 48 }, (_, i) => {
                          const port = `Gi1/0/${i + 1}`;
                          const checked = group.members.some(
                            (m) => m.switchId === sw.id && m.port === port,
                          );
                          return (
                            <label key={port} className="port-check">
                              <input
                                type="checkbox"
                                checked={checked}
                                onChange={() => toggleMember(gi, sw.id, port)}
                              />
                              {port}
                            </label>
                          );
                        })}
                      </div>
                    </div>
                  ))}
                {config.switches.every((sw) => sw.ip.trim() === '') && (
                  <p className="config-message error">
                    Configure switch credentials before assigning ports.
                  </p>
                )}
              </div>
            </div>
          );
        })}

        <div className="config-actions">
          <button type="button" className="btn btn-primary" onClick={addGroup}>
            Add group
          </button>
          <button
            type="button"
            className="btn btn-primary"
            disabled={saving}
            onClick={() => void handleSave()}
          >
            {saving ? 'Saving…' : 'Save groups'}
          </button>
        </div>
        {dirty && (
          <p className="vlan-hint">Save changes before applying VLAN to new or edited groups.</p>
        )}
        {message && (
          <p className={`config-message ${message.includes('saved') ? 'success' : 'error'}`}>
            {message}
          </p>
        )}
      </div>
    </div>
  );
}
