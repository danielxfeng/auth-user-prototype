<script lang="ts">
	import { toastStore } from '$lib/stores';
	import type { ToastType } from '$lib/stores';
	import { fly, fade } from 'svelte/transition';
	import { cn } from '$lib/utils/cn';

	const color: Record<ToastType, string> = {
		success: 'bg-green-500/50 border-green-700',
		error: 'bg-red-500/50 border-red-700',
		info: 'bg-blue-500/50 border-blue-700'
	};
</script>

{#if $toastStore}
	<div
		class={cn(
			'fixed top-4 left-1/2 -translate-x-1/2 rounded-lg border px-4 py-2 text-white backdrop-blur-sm',
			color[$toastStore.type]
		)}
		in:fly={{ y: -10, duration: 200 }}
		out:fade={{ duration: 150 }}
	>
		<span>{$toastStore.message}</span>
	</div>
{/if}
