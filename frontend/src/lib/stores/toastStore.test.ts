import { get } from 'svelte/store';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { toastStore } from './toastStore.js';

describe('toastStore', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		toastStore.clear();
	});

	afterEach(() => {
		vi.runOnlyPendingTimers();
		vi.useRealTimers();
		toastStore.clear();
	});

	it('shows a toast and auto clears after duration', () => {
		toastStore.show('Hello', 'success', 500);

		expect(get(toastStore)).toEqual({
			message: 'Hello',
			type: 'success',
			duration: 500
		});

		vi.advanceTimersByTime(500);

		expect(get(toastStore)).toBeNull();
	});

	it('clears any existing toast before showing a new one', () => {
		toastStore.show('First', 'info', 1_000);
		vi.advanceTimersByTime(400);

		toastStore.show('Second', 'error', 800);

		expect(get(toastStore)).toEqual({
			message: 'Second',
			type: 'error',
			duration: 800
		});

		vi.advanceTimersByTime(800);

		expect(get(toastStore)).toBeNull();
	});

	it('clear() cancels timers and removes the toast immediately', () => {
		toastStore.show('Will clear', 'info', 1_000);
		toastStore.clear();

		vi.advanceTimersByTime(1_000);

		expect(get(toastStore)).toBeNull();
	});
});
