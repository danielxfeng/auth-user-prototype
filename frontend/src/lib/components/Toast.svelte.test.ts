import { afterEach, describe, expect, it } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { page } from 'vitest/browser';

import { toastStore } from '$lib/stores/toastStore.js';
import Toast from './Toast.svelte';

describe('Toast.svelte', () => {
	afterEach(() => {
		toastStore.clear();
	});

	it('renders a toast message when the store has a value', async () => {
		render(Toast);

		toastStore.show('Hello world', 'success', 500);

		const toast = page.getByText('Hello world');
		await expect.element(toast).toBeInTheDocument();
	});

	it('hides after the toast duration elapses', async () => {
		render(Toast);

		toastStore.show('Short lived', 'info', 20);

		const toast = page.getByText('Short lived');
		await expect.element(toast).toBeInTheDocument();

		await new Promise((resolve) => setTimeout(resolve, 60));

		await expect.element(toast).not.toBeInTheDocument();
	});
});
