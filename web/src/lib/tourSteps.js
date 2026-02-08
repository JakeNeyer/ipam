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
