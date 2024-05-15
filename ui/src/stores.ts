import { writable } from 'svelte/store';

export interface ModalState {
    showModal: boolean;
    data: any;
}

export const modalStore = writable<ModalState>({ showModal: false, data: {} });

export const shouldRefresh = writable(false);