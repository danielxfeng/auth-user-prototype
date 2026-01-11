<script lang="ts">
	import { browser } from '$app/environment';
	import CardTabs from './CardTabs.svelte';
	import { Root as CardRoot } from '$lib/components/ui/card/index.js';

	let { children } = $props();
</script>

<div class="flex w-full max-w-4xl flex-col items-center justify-center gap-4 p-6">
	<div class="flex w-full justify-start">
		<a
			class="mb-6 text-start text-2xl font-semibold hover:text-foreground/80 lg:text-3xl xl:text-4xl"
			href="/"
		>
			<h1>User Area</h1>
		</a>
	</div>
	{#if browser}
		<div class="w-full max-w-2xl">
			<CardTabs />
		</div>
	{/if}
	<CardRoot class="flex w-full max-w-2xl justify-center">
		{@render children()}
	</CardRoot>
</div>
