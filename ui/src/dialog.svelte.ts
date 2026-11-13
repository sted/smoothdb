// The confirmation dialog of the UI, in place of the browser's confirm().
// One ConfirmDialog in App shows the current request.

export interface DialogRequest {
    title: string;
    message?: string;
    confirmLabel?: string;
    cancelLabel?: string;
    // a destructive action: red button, and the focus starts on Cancel
    danger?: boolean;
    resolve: (ok: boolean) => void;
}

export const dialog = $state<{ current: DialogRequest | null }>({ current: null });

export function confirmDialog(request: Omit<DialogRequest, 'resolve'>): Promise<boolean> {
    // a request that is still open is cancelled
    dialog.current?.resolve(false);
    return new Promise((resolve) => {
        dialog.current = { ...request, resolve };
    });
}

export function closeDialog(ok: boolean) {
    const current = dialog.current;
    dialog.current = null;
    current?.resolve(ok);
}
