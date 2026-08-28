'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { detectPortChanges, portChangeKey, PortChangeField } from '@/lib/portChanges';
import { ApplyStatus, RuntimeSnapshot, WSMessage } from '@/lib/types';

export type ActionFeedbackType = 'pending' | 'success' | 'error' | 'info';

export interface ActionFeedback {
  type: ActionFeedbackType;
  message: string;
}

export interface ApplyState {
  status: ApplyStatus;
  message?: string;
}

export interface RemoteHighlight {
  fields: PortChangeField[];
}

const HIGHLIGHT_MS = 6000;

function portApplyKey(switchId: string, port: string): string {
  return `port:${switchId}:${port}`;
}

function groupApplyKey(groupId: string): string {
  return `group:${groupId}`;
}

function wsUrl(): string {
  if (typeof window === 'undefined') return '';
  const env = process.env.NEXT_PUBLIC_WS_URL;
  if (env) return env;
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  if (window.location.port === '3000') {
    return `${proto}//${window.location.hostname}:8080/ws`;
  }
  return `${proto}//${window.location.host}/ws`;
}

function isOwnPortChange(applyStates: Record<string, ApplyState>, switchId: string, port: string): boolean {
  const key = portApplyKey(switchId, port);
  const state = applyStates[key]?.status;
  return state === 'pending' || state === 'success';
}

export function useWebSocket() {
  const [connected, setConnected] = useState(false);
  const [snapshot, setSnapshot] = useState<RuntimeSnapshot>({
    switches: [],
    portGroups: [],
    discoveredVlans: [],
    selectableVlans: [],
  });
  const [feedback, setFeedback] = useState<ActionFeedback | null>(null);
  const [applyStates, setApplyStates] = useState<Record<string, ApplyState>>({});
  const [remoteHighlights, setRemoteHighlights] = useState<Record<string, RemoteHighlight>>({});
  const wsRef = useRef<WebSocket | null>(null);
  const feedbackTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const clearStateTimers = useRef<Record<string, ReturnType<typeof setTimeout>>>({});
  const clearHighlightTimers = useRef<Record<string, ReturnType<typeof setTimeout>>>({});
  const lastPendingKey = useRef<string | null>(null);
  const prevSnapshotRef = useRef<RuntimeSnapshot | null>(null);
  const applyStatesRef = useRef(applyStates);

  useEffect(() => {
    applyStatesRef.current = applyStates;
  }, [applyStates]);

  const showFeedback = useCallback((next: ActionFeedback) => {
    if (feedbackTimer.current) clearTimeout(feedbackTimer.current);
    setFeedback(next);
    if (next.type !== 'pending') {
      feedbackTimer.current = setTimeout(() => setFeedback(null), 5000);
    }
  }, []);

  const setApplyState = useCallback((key: string, state: ApplyState | null) => {
    if (clearStateTimers.current[key]) {
      clearTimeout(clearStateTimers.current[key]);
      delete clearStateTimers.current[key];
    }

    setApplyStates((current) => {
      const next = { ...current };
      if (state) {
        next[key] = state;
      } else {
        delete next[key];
      }
      return next;
    });

    if (state?.status === 'success') {
      clearStateTimers.current[key] = setTimeout(() => {
        setApplyStates((current) => {
          if (current[key]?.status !== 'success') return current;
          const next = { ...current };
          delete next[key];
          return next;
        });
        delete clearStateTimers.current[key];
      }, 4000);
    }
  }, []);

  const markRemotePorts = useCallback(
    (entries: { switchId: string; port: string; fields: PortChangeField[] }[]) => {
      if (entries.length === 0) return;

      setRemoteHighlights((current) => {
        const next = { ...current };
        for (const entry of entries) {
          next[portChangeKey(entry)] = { fields: entry.fields };
        }
        return next;
      });

      for (const entry of entries) {
        const key = portChangeKey(entry);
        if (clearHighlightTimers.current[key]) {
          clearTimeout(clearHighlightTimers.current[key]);
        }
        clearHighlightTimers.current[key] = setTimeout(() => {
          setRemoteHighlights((current) => {
            if (!current[key]) return current;
            const next = { ...current };
            delete next[key];
            return next;
          });
          delete clearHighlightTimers.current[key];
        }, HIGHLIGHT_MS);
      }
    },
    [],
  );

  const highlightRemoteChanges = useCallback(
    (changes: ReturnType<typeof detectPortChanges>) => {
      const external = changes.filter(
        (change) => !isOwnPortChange(applyStatesRef.current, change.switchId, change.port),
      );
      if (external.length === 0) return;

      markRemotePorts(
        external.map((change) => ({
          switchId: change.switchId,
          port: change.port,
          fields: change.fields,
        })),
      );
    },
    [markRemotePorts],
  );

  const handleSnapshotUpdate = useCallback(
    (next: RuntimeSnapshot) => {
      const prev = prevSnapshotRef.current;
      if (prev && prev.switches.length > 0) {
        const changes = detectPortChanges(prev, next);
        if (changes.length > 0) {
          highlightRemoteChanges(changes);
        }
      }
      prevSnapshotRef.current = next;
      setSnapshot(next);
    },
    [highlightRemoteChanges],
  );

  const failPending = useCallback(
    (message: string) => {
      const key = lastPendingKey.current;
      if (key) {
        setApplyState(key, { status: 'error', message });
        lastPendingKey.current = null;
      }
      showFeedback({ type: 'error', message });
    },
    [setApplyState, showFeedback],
  );

  useEffect(() => {
    let closed = false;
    let retryTimer: ReturnType<typeof setTimeout>;

    const connect = () => {
      const ws = new WebSocket(wsUrl());
      wsRef.current = ws;

      ws.onopen = () => setConnected(true);
      ws.onclose = () => {
        setConnected(false);
        if (!closed) retryTimer = setTimeout(connect, 2000);
      };
      ws.onerror = () => ws.close();

      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data) as WSMessage;
          if (msg.type === 'state-update' && msg.switches) {
            handleSnapshotUpdate({
              switches: msg.switches,
              portGroups: msg.portGroups ?? [],
              discoveredVlans: msg.discoveredVlans ?? [],
              selectableVlans: msg.selectableVlans ?? [],
            });
            return;
          }
          if (msg.type === 'vlan-applying') {
            if (msg.switchId && msg.port) {
              const key = portApplyKey(msg.switchId, msg.port);
              if (key !== lastPendingKey.current) {
                setApplyState(key, { status: 'pending' });
              }
            }
            return;
          }
          if (msg.type === 'group-vlan-applying') {
            if (msg.groupId) {
              const key = groupApplyKey(msg.groupId);
              if (key !== lastPendingKey.current) {
                setApplyState(key, { status: 'pending' });
              }
            }
            return;
          }
          if (msg.type === 'vlan-changed') {
            const key =
              msg.switchId && msg.port ? portApplyKey(msg.switchId, msg.port) : lastPendingKey.current;
            const isOwn = key !== null && key === lastPendingKey.current;

            if (key && msg.ok) {
              if (isOwn) {
                setApplyState(key, { status: 'success' });
                showFeedback({
                  type: 'success',
                  message: `Applied VLAN ${msg.vlan} on ${msg.port}`,
                });
                lastPendingKey.current = null;
              } else if (msg.switchId && msg.port) {
                const key = portApplyKey(msg.switchId, msg.port);
                setApplyState(key, { status: 'success' });
                markRemotePorts([
                  { switchId: msg.switchId, port: msg.port, fields: ['vlan'] },
                ]);
                showFeedback({
                  type: 'info',
                  message: `${msg.port} → VLAN ${msg.vlan}`,
                });
              }
            } else if (key && !msg.ok && isOwn) {
              setApplyState(key, { status: 'error', message: 'VLAN apply failed' });
              showFeedback({ type: 'error', message: 'VLAN apply failed' });
              lastPendingKey.current = null;
            }
            return;
          }
          if (msg.type === 'group-vlan-changed') {
            const key = msg.groupId ? groupApplyKey(msg.groupId) : lastPendingKey.current;
            const isOwn = key !== null && key === lastPendingKey.current;

            if (key && msg.ok) {
              if (isOwn) {
                setApplyState(key, { status: 'success' });
                showFeedback({
                  type: 'success',
                  message: `Applied VLAN ${msg.vlan} to group`,
                });
                lastPendingKey.current = null;
              } else {
                if (msg.groupId) {
                  setApplyState(groupApplyKey(msg.groupId), { status: 'success' });
                }
                const entries =
                  msg.results
                    ?.filter((r) => r.ok)
                    .map((r) => ({
                      switchId: r.switchId,
                      port: r.port,
                      fields: ['vlan' as PortChangeField],
                    })) ?? [];
                markRemotePorts(entries);
                showFeedback({
                  type: 'info',
                  message: `Group → VLAN ${msg.vlan}`,
                });
              }
            } else if (key && !msg.ok && isOwn) {
              setApplyState(key, {
                status: 'error',
                message: 'Some ports in the group failed to update',
              });
              showFeedback({
                type: 'error',
                message: 'Some ports in the group failed to update',
              });
              lastPendingKey.current = null;
            }
            return;
          }
          if (msg.type === 'error' && msg.message) {
            failPending(msg.message);
          }
        } catch {
          // ignore malformed messages
        }
      };
    };

    connect();

    return () => {
      closed = true;
      clearTimeout(retryTimer);
      if (feedbackTimer.current) clearTimeout(feedbackTimer.current);
      Object.values(clearStateTimers.current).forEach(clearTimeout);
      Object.values(clearHighlightTimers.current).forEach(clearTimeout);
      wsRef.current?.close();
    };
  }, [failPending, handleSnapshotUpdate, markRemotePorts, setApplyState, showFeedback]);

  const send = useCallback((msg: WSMessage): boolean => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(msg));
      return true;
    }
    return false;
  }, []);

  const setPortVlan = useCallback(
    (switchId: string, port: string, vlan: number) => {
      const key = portApplyKey(switchId, port);
      lastPendingKey.current = key;
      setApplyState(key, { status: 'pending' });
      showFeedback({
        type: 'pending',
        message: `Applying VLAN ${vlan} on ${port}…`,
      });
      if (!send({ type: 'set-vlan', switchId, port, vlan })) {
        failPending('Not connected to backend');
      }
    },
    [failPending, send, setApplyState, showFeedback],
  );

  const setGroupVlan = useCallback(
    (groupId: string, vlan: number) => {
      const key = groupApplyKey(groupId);
      lastPendingKey.current = key;
      setApplyState(key, { status: 'pending' });
      showFeedback({
        type: 'pending',
        message: `Applying VLAN ${vlan} to group…`,
      });
      if (!send({ type: 'set-group-vlan', groupId, vlan })) {
        failPending('Not connected to backend');
      }
    },
    [failPending, send, setApplyState, showFeedback],
  );

  const refreshSwitch = useCallback(
    (switchId: string) => {
      send({ type: 'refresh-switch', switchId });
    },
    [send],
  );

  const getPortApplyStatus = useCallback(
    (switchId: string, port: string): ApplyStatus | null =>
      applyStates[portApplyKey(switchId, port)]?.status ?? null,
    [applyStates],
  );

  const getGroupApplyStatus = useCallback(
    (groupId: string): ApplyStatus | null =>
      applyStates[groupApplyKey(groupId)]?.status ?? null,
    [applyStates],
  );

  const getPortRemoteChange = useCallback(
    (switchId: string, port: string): RemoteHighlight | null =>
      remoteHighlights[portChangeKey({ switchId, port })] ?? null,
    [remoteHighlights],
  );

  return {
    connected,
    snapshot,
    feedback,
    setPortVlan,
    setGroupVlan,
    refreshSwitch,
    getPortApplyStatus,
    getGroupApplyStatus,
    getPortRemoteChange,
  };
}
