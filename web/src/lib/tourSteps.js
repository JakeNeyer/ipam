/**
 * Tour step definitions. Each step can target a DOM element via data-tour="<targetId>"
 * or use targetId null for a centered step (e.g. welcome).
 */
export const tourSteps = [
  {
    id: 'welcome',
    targetId: null,
    title: 'Welcome to IPAM',
    body: 'This short tour will show you the main areas of the app. You can skip anytime.',
  },
  {
    id: 'nav-dashboard',
    targetId: 'tour-nav-dashboard',
    title: 'Dashboard',
    body: 'View block utilization, resource stats, and quick links to environments and networks.',
  },
  {
    id: 'nav-environments',
    targetId: 'tour-nav-environments',
    title: 'Environments',
    body: 'Create and manage environments (e.g. production, staging). Each environment can have multiple blocks.',
  },
  {
    id: 'nav-networks',
    targetId: 'tour-nav-networks',
    title: 'Networks',
    body: 'Browse blocks and allocations, filter by environment, and create new blocks or IP allocations.',
  },
  {
    id: 'nav-network-advisor',
    targetId: 'tour-nav-network-advisor',
    title: 'Network Advisor',
    body: 'Get AI-assisted suggestions for subnet sizing and placement.',
  },
  {
    id: 'nav-subnet-calculator',
    targetId: 'tour-nav-subnet-calculator',
    title: 'Subnet calculator',
    body: 'Calculate subnets, CIDR ranges, and IP availability.',
  },
  {
    id: 'nav-reserved-blocks',
    targetId: 'tour-nav-reserved-blocks',
    title: 'Reserved blocks',
    body: 'View and manage reserved CIDR blocks that are excluded from allocation (admin only).',
  },
  {
    id: 'nav-integrations',
    targetId: 'tour-nav-integrations',
    title: 'Integrations',
    body: 'Connect cloud providers to sync pools and blocks from AWS or other sources (admin only).',
  },
  {
    id: 'nav-admin',
    targetId: 'tour-nav-admin',
    title: 'Admin',
    body: 'Manage users, API tokens, organizations, and app settings (admin only).',
  },
  {
    id: 'nav-global-admin-dashboard',
    targetId: 'tour-nav-global-admin-dashboard',
    title: 'Global Admin Dashboard',
    body: 'Switch organizations and view cross-org overview (global admin only).',
  },
  {
    id: 'command-palette',
    targetId: 'tour-command-palette',
    title: 'Command palette',
    body: 'Press ⌘K (or Ctrl+K) anytime to search and jump to environments, blocks, or create new resources.',
  },
  {
    id: 'nav-settings',
    targetId: 'tour-nav-settings',
    title: 'Settings',
    body: 'Access the user guide, theme, and sign out.',
  },
  {
    id: 'done',
    targetId: null,
    title: "You're all set",
    body: 'Explore the Dashboard and Networks to get started. Need help? Check the User guide in Settings.',
  },
]
