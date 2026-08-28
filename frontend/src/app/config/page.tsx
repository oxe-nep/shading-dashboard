'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { fetchConfig, saveConfig } from '@/lib/api';
import { AppConfig } from '@/lib/types';

export default function ConfigPage() {
  const router = useRouter();
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    fetchConfig().then(setConfig).catch(console.error);
  }, []);

  const updateSwitch = (index: number, field: keyof AppConfig['switches'][0], value: string | number) => {
    if (!config) return;
    const switches = [...config.switches];
    switches[index] = { ...switches[index], [field]: value };
    setConfig({ ...config, switches });
  };

  const handleSave = async () => {
    if (!config) return;
    if (config.switches.some((s) => !s.name.trim())) {
      setMessage('Each switch must have a name.');
      return;
    }
    setSaving(true);
    setMessage('');
    try {
      await saveConfig(config);
      router.push('/');
    } catch (err) {
      setMessage(String(err));
      setSaving(false);
    }
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
      <div className="config-section">
        <h2>Switches ({config.switches.length})</h2>
        <div className="config-table-wrapper">
          <table className="config-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>IP address</th>
                <th>Username</th>
                <th>Password</th>
                <th>Port</th>
              </tr>
            </thead>
            <tbody>
              {config.switches.map((sw, index) => (
                <tr key={sw.id}>
                  <td>
                    <input
                      type="text"
                      value={sw.name}
                      onChange={(e) => updateSwitch(index, 'name', e.target.value)}
                    />
                  </td>
                  <td>
                    <input
                      type="text"
                      value={sw.ip}
                      onChange={(e) => updateSwitch(index, 'ip', e.target.value)}
                    />
                  </td>
                  <td>
                    <input
                      type="text"
                      value={sw.username}
                      onChange={(e) => updateSwitch(index, 'username', e.target.value)}
                    />
                  </td>
                  <td>
                    <input
                      type="password"
                      value={sw.password}
                      onChange={(e) => updateSwitch(index, 'password', e.target.value)}
                    />
                  </td>
                  <td>
                    <input
                      type="number"
                      value={sw.port ?? 830}
                      onChange={(e) => updateSwitch(index, 'port', parseInt(e.target.value, 10) || 830)}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {message && <p className="config-message error">{message}</p>}
        <div className="config-actions">
          <button type="button" className="btn btn-primary" disabled={saving} onClick={() => void handleSave()}>
            {saving ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  );
}
