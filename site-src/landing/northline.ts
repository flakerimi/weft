// Northline, a fibre ISP in Prishtina that does not exist: the data the
// landing page's pictures of an app draw. Every name, port and number here is
// made up, and none comes from a real network.
import type { MenuItem, NavItem, TableColumn, Tone } from '@weft/ui'

export interface Device {
  id: string
  /** The subscriber the ONT serves. */
  name: string
  /** OLT shelf, slot and port. */
  port: string
  /** Received optical power, or a dash when it is offline. */
  signal: string
  status: 'online' | 'weak' | 'offline'
}

export const devices: Device[] = [
  { id: 'krasniqi', name: 'Krasniqi A.', port: '1/2/4', signal: '−19.4 dBm', status: 'online' },
  { id: 'berisha', name: 'Berisha L.', port: '1/3/1', signal: '−24.9 dBm', status: 'weak' },
  { id: 'gashi', name: 'Gashi M.', port: '1/3/6', signal: '—', status: 'offline' },
  { id: 'hoxha', name: 'Hoxha E.', port: '1/1/7', signal: '−18.1 dBm', status: 'online' },
  { id: 'morina', name: 'Morina D.', port: '1/4/3', signal: '−20.7 dBm', status: 'online' },
]

/** The device whose menu the pictures draw open. */
export const opened = 'berisha'

export const statusOf: Record<Device['status'], { label: string; tone: Tone }> = {
  online: { label: 'Online', tone: 'success' },
  weak: { label: 'Weak', tone: 'warning' },
  offline: { label: 'Offline', tone: 'danger' },
}

export const columns: TableColumn<Device>[] = [
  { key: 'name', label: 'Device', rowHeader: true },
  { key: 'port', label: 'Port' },
  { key: 'signal', label: 'Signal' },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: 'Actions' },
]

export const nav: NavItem[] = [
  { label: 'Overview', icon: 'layout-dashboard' },
  { label: 'Devices', icon: 'router', active: true },
  { label: 'Customers', icon: 'users' },
  { label: 'Traffic', icon: 'activity' },
  { label: 'Tickets', icon: 'ticket' },
  { label: 'Settings', icon: 'settings' },
]

/** A device's menu, headed by its name. */
export const deviceMenu = (device: Device): MenuItem[][] => [
  [
    { type: 'label', label: device.name },
    { label: 'Open device', icon: 'square-arrow-out-up-right' },
    { label: 'Reboot', icon: 'rotate-cw', kbds: ['command', 'R'] },
    { label: 'Move port…', icon: 'arrow-right-left' },
  ],
  [{ label: 'Remove', icon: 'trash-2', color: 'danger' }],
]
