import type { UserWithoutTokenResponse, UserWithTokenResponse } from '$lib/schemas/types';
import { writable } from 'svelte/store';

const STORAGE_USER = 'auth_user';
export const STORAGE_TOKEN = 'auth_token';

type UserStore = {
	user: UserWithoutTokenResponse | null;
	token: string | null;
};

const saveToLocalStorage = (state: UserStore): void => {
	if (!state.user || !state.token) return;
	if (typeof window === 'undefined') return;

	localStorage.setItem(STORAGE_USER, JSON.stringify(state.user));
	localStorage.setItem(STORAGE_TOKEN, state.token);
};

const removeFromLocalStorage = () => {
	if (typeof window === 'undefined') return;

	localStorage.removeItem(STORAGE_USER);
	localStorage.removeItem(STORAGE_TOKEN);
};

const getFromLocalStorage = (): UserStore => {
	if (typeof window === 'undefined') return { user: null, token: null };

	try {
		const userStr = localStorage.getItem(STORAGE_USER);
		const token = localStorage.getItem(STORAGE_TOKEN);
		const user = userStr ? (JSON.parse(userStr) as UserWithoutTokenResponse) : null;

		if (!user || !token) return { user: null, token: null };
		return { user, token };
	} catch {
		return { user: null, token: null };
	}
};

const { subscribe, set } = writable<UserStore>({ user: null, token: null });

export const userStore = {
	subscribe,

	hydrate() {
		set(getFromLocalStorage());
	},

	login(user: UserWithTokenResponse) {
		const { token, ...userWithoutToken } = user;
		const nextState: UserStore = {
			user: userWithoutToken,
			token
		};
		set(nextState);
		saveToLocalStorage(nextState);
	},

	logout() {
		set({ user: null, token: null });
		removeFromLocalStorage();
	}
};
