import { writable } from 'svelte/store';

export type ToastType = 'success' | 'error' | 'info';

export type Toast = {
	message: string;
	type: ToastType;
	duration: number;
};

const { subscribe, set } = writable<Toast | null>(null);

let timer: NodeJS.Timeout | null = null;

export const toastStore = {
	subscribe,

	show(message: string, type: ToastType = 'info', duration = 3000) {
		if (timer) {
			clearTimeout(timer);
			timer = null;
		}

		set({ message, type, duration });

		timer = setTimeout(() => {
			set(null);
			timer = null;
		}, duration);
	},

	clear() {
		if (timer) {
			clearTimeout(timer);
			timer = null;
		}
		set(null);
	}
};
