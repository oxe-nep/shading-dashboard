import { AppConfig } from './types';

function normalizeConfig(config: AppConfig): AppConfig {
  return {
    ...config,
    portGroups: config.portGroups ?? [],
    allowedVlans: config.allowedVlans ?? [],
  };
}

export async function fetchConfig(): Promise<AppConfig> {
  const res = await fetch('/api/config');
  if (!res.ok) throw new Error('Failed to load config');
  return normalizeConfig(await res.json());
}

export async function saveConfig(config: AppConfig): Promise<AppConfig> {
  const res = await fetch('/api/config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || 'Failed to save config');
  }
  return normalizeConfig(await res.json());
}

export async function savePortGroups(portGroups: AppConfig['portGroups']): Promise<AppConfig> {
  const res = await fetch('/api/port-groups', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(portGroups),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || 'Failed to save port groups');
  }
  return normalizeConfig(await res.json());
}
