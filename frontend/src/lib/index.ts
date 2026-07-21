import { Library, Download, DownloadCloud, Activity, Clock, Settings } from 'lucide-svelte';

export const navigationItems = [
    { href: '/library', label: 'Library', icon: Library },
    { href: '/wanted', label: 'Wanted', icon: Download },
    { href: '/queue', label: 'Queue', icon: DownloadCloud },
    { href: '/activity', label: 'Activity', icon: Activity },
    { href: '/history', label: 'History', icon: Clock },
    { href: '/settings', label: 'Settings', icon: Settings }
];
