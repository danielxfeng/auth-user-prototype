<script lang="ts">
	import { browser } from '$app/environment';
	import CardTabs from './CardTabs.svelte';

	let { children } = $props();
</script>

<div class="flex w-full max-w-2xl flex-col items-center justify-center">
	{#if browser}
		<CardTabs />
	{/if}
	<div class="w-full rounded-xl border border-border p-6">
		{@render children()}
	</div>
</div>
