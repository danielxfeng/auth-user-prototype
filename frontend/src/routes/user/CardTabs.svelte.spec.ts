import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { page as browserPage } from 'vitest/browser';

vi.mock('$app/state', () => {
	const page = { url: new URL('http://localhost/login/') };
	const setPage = (url: string) => {
		page.url = new URL(url);
	};
	return { page, __setPage: setPage };
});

vi.mock('$lib/stores', async () => {
	const { writable } = await import('svelte/store');
	const userStore = writable<{ user: unknown | null; token: string | null }>({
		user: null,
		token: null
	});
	const __setUser = (user: unknown) => userStore.set({ user, token: user ? 'token' : null });
	return { userStore, __setUser };
});

let setPage: (url: string) => void;
let setUser: (user: unknown) => void;

beforeAll(async () => {
	const state = (await import('$app/state')) as unknown as {
		__setPage: (url: string) => void;
	};
	const stores = (await import('$lib/stores')) as unknown as {
		__setUser: (user: unknown) => void;
	};
	setPage = state.__setPage;
	setUser = stores.__setUser;
});

import CardTabs from './CardTabs.svelte';

describe('CardTabs', () => {
	beforeEach(() => {
		setUser(null);
		setPage('http://localhost/login/');
	});

	afterEach(() => {
		vi.clearAllMocks();
	});

	it('shows guest links with the active tab styled', async () => {
		render(CardTabs);

		const login = browserPage.getByText('Login');
		const register = browserPage.getByText('Register');

		await expect.element(login).toBeInTheDocument();
		await expect.element(register).toBeInTheDocument();
		await expect.element(login).toHaveClass('bg-primary');
		await expect.element(register).toHaveClass('bg-background');
	});

	it('shows logged-in links and highlights the active one', async () => {
		setUser({ username: 'alice' });
		setPage('http://localhost/friends/');

		render(CardTabs);

		const profileLink = browserPage.getByText('Profile');
		const settingsLink = browserPage.getByText('Settings');
		const friendsLink = browserPage.getByText('Friends');

		await expect.element(profileLink).toBeInTheDocument();
		await expect.element(settingsLink).toBeInTheDocument();
		await expect.element(friendsLink).toBeInTheDocument();
		await expect.element(friendsLink).toHaveClass('bg-primary');
		await expect.element(profileLink).toHaveClass('bg-background');
		await expect.element(settingsLink).toHaveClass('bg-background');
	});
});
