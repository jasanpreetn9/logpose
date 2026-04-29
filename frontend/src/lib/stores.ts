import { writable } from 'svelte/store';

export const arcs = writable<UnifiedArc[]>([]);
export const sidebarOpen = writable(false);
export const downloading = writable<Set<string>>(new Set());
export const activity = writable<ActivityEvent[]>([]);
export const historyEvents = writable<ActivityEvent[]>([]);
