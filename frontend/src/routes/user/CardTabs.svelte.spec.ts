import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { page as browserPage } from 'vitest/browser';

// Extend module typings for mocked helpers
declare module '$app/state' {
	export function __setPage(url: string): void;
}
declare module '$lib/stores' {
	export function __setUser(user: unknown): void;
}

vi.mock('$app/state', async () => {
	const page = { url: new URL('http://localhost/login/') };
	return {
		page,
		__setPage: (url: string) => {
			page.url = new URL(url);
		}
	};
});

vi.mock('$lib/stores', async () => {
	const { writable } = await import('svelte/store');
	type MockUserStore = { user: unknown | null; token: string | null };
	const userStore = writable<MockUserStore>({ user: null, token: null });
	return {
		userStore,
		__setUser: (user: unknown) => userStore.set({ user, token: user ? 'token' : null })
	};
});

import { __setPage } from '$app/state';
import { __setUser } from '$lib/stores';
import CardTabs from './CardTabs.svelte';

describe('CardTabs', () => {
	beforeEach(() => {
		__setUser(null);
		__setPage('http://localhost/login/');
	});

	afterEach(() => {
		vi.clearAllMocks();
	});

	it('shows guest links when user is not logged in', async () => {
		render(CardTabs);

		await expect.element(browserPage.getByText('Login')).toBeInTheDocument();
		await expect.element(browserPage.getByText('Register')).toBeInTheDocument();
	});

	it('shows logged-in links and highlights the active one', async () => {
		__setUser({ username: 'alice' });
		__setPage('http://localhost/profile/');

		render(CardTabs);

		const profileLink = browserPage.getByText('Profile');
		const settingsLink = browserPage.getByText('Settings');

		await expect.element(profileLink).toBeInTheDocument();
		await expect.element(settingsLink).toBeInTheDocument();
		await expect.element(profileLink).toHaveClass('bg-accent');
		await expect.element(profileLink).toHaveClass('text-primary');
		await expect.element(settingsLink).not.toHaveClass('bg-accent');
	});
});
