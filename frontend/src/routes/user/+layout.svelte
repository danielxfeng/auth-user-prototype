<script lang="ts">
	import { browser } from '$app/environment';
	import CardTabs from './CardTabs.svelte';
	import { Root as CardRoot } from '$lib/components/ui/card/index.js';

	let { children } = $props();
</script>

<div class="flex w-full max-w-2xl flex-col justify-center gap-4 p-6">
	<h1 class="mb-6 text-2xl font-semibold lg:text-3xl xl:text-4xl">User Area</h1>
	{#if browser}
		<CardTabs />
	{/if}
	<CardRoot class="flex w-full justify-center">
		{@render children()}
	</CardRoot>
</div>
