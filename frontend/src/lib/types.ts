export interface PortGroupMember {
  switchId: string;
  port: string;
}

export interface PortGroupConfig {
  id: string;
  name: string;
  members: PortGroupMember[];
}

export interface SwitchConfig {
  id: string;
  name: string;
  ip: string;
  username: string;
  password: string;
  port?: number;
}

export interface AppConfig {
  switches: SwitchConfig[];
  portGroups: PortGroupConfig[];
  allowedVlans?: number[];
}

export interface VlanInfo {
  id: number;
  name?: string;
  label: string;
  title: string;
}

export interface PortState {
  name: string;
  operState: string;
  accessVlan: number | null;
  accessVlanLabel?: string;
  accessVlanTitle?: string;
  adminDown: boolean;
  description?: string;
}

export interface SwitchRuntimeState {
  id: string;
  name: string;
  ip: string;
  online: boolean;
  polling: boolean;
  lastPollAt: string | null;
  lastSuccessAt: string | null;
  ports: PortState[];
  vlans?: VlanInfo[];
  lastError?: string;
}

export interface PortGroupRuntime {
  id: string;
  name: string;
  members: PortGroupMember[];
  currentVlan: number | null;
  vlanLabel: string;
  mixed: boolean;
}

export interface PortApplyResult {
  switchId: string;
  port: string;
  ok: boolean;
  error?: string;
}

export type ApplyStatus = 'pending' | 'success' | 'error';

export interface RuntimeSnapshot {
  switches: SwitchRuntimeState[];
  portGroups: PortGroupRuntime[];
  discoveredVlans?: VlanInfo[];
  selectableVlans?: VlanInfo[];
}

export interface WSMessage {
  type: string;
  switches?: SwitchRuntimeState[];
  portGroups?: PortGroupRuntime[];
  discoveredVlans?: VlanInfo[];
  selectableVlans?: VlanInfo[];
  switchId?: string;
  groupId?: string;
  port?: string;
  vlan?: number;
  ok?: boolean;
  message?: string;
  results?: PortApplyResult[];
}
