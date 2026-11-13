// Short messages at the bottom right of the page.

export interface Toast {
    id: number;
    kind: 'error' | 'info';
    text: string;
    detail?: string;
}

export const toasts: Toast[] = $state([]);
let nextId = 1;

export function dismiss(id: number) {
    const i = toasts.findIndex((t) => t.id === id);
    if (i >= 0) toasts.splice(i, 1);
}

export function showToast(kind: Toast['kind'], text: string, detail?: string) {
    const id = nextId++;
    toasts.push({ id, kind, text, detail });
    // errors stay until the user closes them
    if (kind === 'info') setTimeout(() => dismiss(id), 4000);
}

export const showError = (text: string, detail?: string) => showToast('error', text, detail);
export const showInfo = (text: string) => showToast('info', text);
