import { PortState, RuntimeSnapshot, SwitchRuntimeState } from '@/lib/types';

export type PortChangeField = 'vlan' | 'description' | 'link';

export interface PortChange {
  switchId: string;
  port: string;
  fields: PortChangeField[];
}

function portKey(switchId: string, port: string): string {
  return `${switchId}:${port}`;
}

function findSwitch(switches: SwitchRuntimeState[], id: string): SwitchRuntimeState | undefined {
  return switches.find((sw) => sw.id === id);
}

function portFieldsChanged(before: PortState, after: PortState): PortChangeField[] {
  const fields: PortChangeField[] = [];
  if (before.accessVlan !== after.accessVlan) {
    fields.push('vlan');
  }
  if ((before.description ?? '') !== (after.description ?? '')) {
    fields.push('description');
  }
  if (before.operState !== after.operState) {
    fields.push('link');
  }
  return fields;
}

export function detectPortChanges(
  before: RuntimeSnapshot,
  after: RuntimeSnapshot,
): PortChange[] {
  const changes: PortChange[] = [];

  for (const sw of after.switches) {
    const prevSw = findSwitch(before.switches, sw.id);
    if (!prevSw) continue;

    const prevByName = new Map(prevSw.ports.map((p) => [p.name, p]));
    for (const port of sw.ports) {
      const prevPort = prevByName.get(port.name);
      if (!prevPort) continue;

      const fields = portFieldsChanged(prevPort, port);
      if (fields.length > 0) {
        changes.push({ switchId: sw.id, port: port.name, fields });
      }
    }
  }

  return changes;
}

export function portChangeKey(change: Pick<PortChange, 'switchId' | 'port'>): string {
  return portKey(change.switchId, change.port);
}

export function formatPortChange(change: PortChange, after: RuntimeSnapshot): string {
  const sw = findSwitch(after.switches, change.switchId);
  const port = sw?.ports.find((p) => p.name === change.port);
  const parts: string[] = [];

  if (change.fields.includes('vlan') && port) {
    const label = port.accessVlanLabel ?? (port.accessVlan != null ? String(port.accessVlan) : '—');
    parts.push(`VLAN → ${label}`);
  }
  if (change.fields.includes('description') && port) {
    parts.push(`desc → ${port.description || '—'}`);
  }
  if (change.fields.includes('link') && port) {
    parts.push(`link → ${port.operState}`);
  }

  const detail = parts.length > 0 ? ` (${parts.join(', ')})` : '';
  return `${change.port}${detail}`;
}
