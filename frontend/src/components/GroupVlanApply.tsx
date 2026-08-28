'use client';

import { useEffect, useState } from 'react';
import { ApplyStatus, PortGroupRuntime, VlanInfo } from '@/lib/types';
import VlanSelect from '@/components/VlanSelect';

interface GroupVlanApplyProps {
  group: PortGroupRuntime;
  vlans: VlanInfo[];
  applyStatus: ApplyStatus | null;
  onApplyVlan: (groupId: string, vlan: number) => void;
  disabled?: boolean;
  saved?: boolean;
}

export default function GroupVlanApply({
  group,
  vlans,
  applyStatus,
  onApplyVlan,
  disabled,
  saved = true,
}: GroupVlanApplyProps) {
  const [vlan, setVlan] = useState('');
  const pending = applyStatus === 'pending';

  useEffect(() => {
    if (applyStatus === 'success') {
      setVlan('');
    }
  }, [applyStatus]);

  return (
    <div
      className={[
        'group-vlan-apply',
        applyStatus ? `apply-state-${applyStatus}` : '',
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <div className="group-vlan-status">
        <span className="label">Current VLAN</span>
        <span className={`badge ${group.mixed ? 'badge-orange' : 'badge-green'}`}>
          {group.vlanLabel}
        </span>
      </div>

      <div className="vlan-set-row group-apply">
        <VlanSelect
          vlans={vlans}
          value={vlan}
          disabled={disabled || pending || !saved}
          onChange={setVlan}
        />
        <button
          type="button"
          className="btn btn-primary"
          disabled={disabled || pending || !vlan || !saved}
          onClick={() => {
            const n = parseInt(vlan, 10);
            if (n > 0) onApplyVlan(group.id, n);
          }}
        >
          {pending ? (
            <>
              <i className="fas fa-spinner fa-spin" /> Applying…
            </>
          ) : (
            'Apply to group'
          )}
        </button>
      </div>

      {!saved && (
        <p className="vlan-hint">Save the group before applying a VLAN.</p>
      )}
    </div>
  );
}
