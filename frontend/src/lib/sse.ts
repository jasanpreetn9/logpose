const SSE_URL = '/api/events';

export function startSSE(onEvent: (ev: ActivityEvent) => void): () => void {
    const source = new EventSource(SSE_URL);

    source.onmessage = (e) => {
        try {
            const ev = JSON.parse(e.data) as ActivityEvent;
            onEvent(ev);
        } catch {
            // ignore malformed frames
        }
    };

    source.onerror = () => {
        // browser will auto-reconnect; no action needed
    };

    return () => source.close();
}
