import { writable } from 'svelte/store'

const STORAGE_KEY = 'ipam-theme'

function getInitialTheme() {
  if (typeof window === 'undefined') return 'dark'
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'light' || stored === 'dark') return stored
  return 'dark'
}

const initial = getInitialTheme()
if (typeof document !== 'undefined') {
  document.documentElement.setAttribute('data-theme', 'wintry')
  document.documentElement.classList.toggle('dark', initial === 'dark')
}

export const theme = writable(initial)

theme.subscribe((value) => {
  if (typeof window === 'undefined') return
  localStorage.setItem(STORAGE_KEY, value)
  document.documentElement.setAttribute('data-theme', 'wintry')
  document.documentElement.classList.toggle('dark', value === 'dark')
})
