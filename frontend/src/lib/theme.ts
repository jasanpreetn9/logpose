import { writable } from 'svelte/store';

export type Theme = 'light' | 'dark' | 'system';

const STORAGE_KEY = 'onepace-theme';

function getInitial(): Theme {
    if (typeof localStorage === 'undefined') return 'system';
    return (localStorage.getItem(STORAGE_KEY) as Theme) ?? 'system';
}

function applyTheme(theme: Theme) {
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    const dark = theme === 'dark' || (theme === 'system' && prefersDark);
    document.documentElement.classList.toggle('dark', dark);
}

export const theme = writable<Theme>(getInitial());

theme.subscribe((t) => {
    if (typeof localStorage === 'undefined') return;
    localStorage.setItem(STORAGE_KEY, t);
    applyTheme(t);
});

// Keep system theme in sync with OS preference changes.
if (typeof window !== 'undefined') {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        theme.update((t) => t); // re-trigger subscribe with current value
    });
}
