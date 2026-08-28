'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { ApplyStatus, RuntimeSnapshot, WSMessage } from '@/lib/types';

export type ActionFeedbackType = 'pending' | 'success' | 'error';

export interface ActionFeedback {
  type: ActionFeedbackType;
  message: string;
}

export interface ApplyState {
  status: ApplyStatus;
  message?: string;
}

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
  // Local Next.js dev — backend on :8080
  if (window.location.port === '3000') {
    return `${proto}//${window.location.hostname}:8080/ws`;
  }
  // Production / k8s — Traefik routes /ws on same host (port is "" on 443/80)
  return `${proto}//${window.location.host}/ws`;
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
  const wsRef = useRef<WebSocket | null>(null);
  const feedbackTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const clearStateTimers = useRef<Record<string, ReturnType<typeof setTimeout>>>({});
  const lastPendingKey = useRef<string | null>(null);

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
            setSnapshot({
              switches: msg.switches,
              portGroups: msg.portGroups ?? [],
              discoveredVlans: msg.discoveredVlans ?? [],
              selectableVlans: msg.selectableVlans ?? [],
            });
            return;
          }
          if (msg.type === 'vlan-changed') {
            const key =
              msg.switchId && msg.port ? portApplyKey(msg.switchId, msg.port) : lastPendingKey.current;
            if (key) {
              if (msg.ok) {
                setApplyState(key, { status: 'success' });
                showFeedback({
                  type: 'success',
                  message: `Applied VLAN ${msg.vlan} on ${msg.port}`,
                });
              } else {
                setApplyState(key, { status: 'error', message: 'VLAN apply failed' });
                showFeedback({ type: 'error', message: 'VLAN apply failed' });
              }
              lastPendingKey.current = null;
            }
            return;
          }
          if (msg.type === 'group-vlan-changed') {
            const key = msg.groupId ? groupApplyKey(msg.groupId) : lastPendingKey.current;
            if (key) {
              if (msg.ok) {
                setApplyState(key, { status: 'success' });
                showFeedback({
                  type: 'success',
                  message: `Applied VLAN ${msg.vlan} to group`,
                });
              } else {
                setApplyState(key, {
                  status: 'error',
                  message: 'Some ports in the group failed to update',
                });
                showFeedback({
                  type: 'error',
                  message: 'Some ports in the group failed to update',
                });
              }
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
      wsRef.current?.close();
    };
  }, [failPending, setApplyState, showFeedback]);

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

  return {
    connected,
    snapshot,
    feedback,
    setPortVlan,
    setGroupVlan,
    refreshSwitch,
    getPortApplyStatus,
    getGroupApplyStatus,
  };
}
