'use client';

import { VlanInfo } from '@/lib/types';

interface VlanSelectProps {
  vlans: VlanInfo[];
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  className?: string;
}

export default function VlanSelect({
  vlans,
  value,
  onChange,
  disabled,
  className,
}: VlanSelectProps) {
  const selectClass = ['vlan-select', className].filter(Boolean).join(' ');

  return (
    <select
      className={selectClass}
      value={value}
      disabled={disabled || vlans.length === 0}
      onChange={(e) => onChange(e.target.value)}
      title={vlans.length === 0 ? 'Configure VLANs on the VLANs page' : undefined}
    >
      <option value="">
        {vlans.length === 0 ? 'No VLANs' : 'Select VLAN…'}
      </option>
      {vlans.map((v) => (
        <option key={v.id} value={String(v.id)} title={v.title}>
          {v.label} ({v.id})
        </option>
      ))}
    </select>
  );
}
