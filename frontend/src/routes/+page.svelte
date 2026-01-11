<script lang="ts">
	import { browser } from '$app/environment';
	import HealthCheck from './HealthCheck.svelte';
	import NavLinks from './NavLinks.svelte';
</script>

<div class="flex flex-1 flex-col items-center justify-center gap-10">
	<div class="flex max-w-4xl flex-col items-center justify-center gap-18 p-4">
		<div class="flex flex-col items-center justify-center gap-2">
			<h1 class="animate-enter text-center text-2xl font-semibold lg:text-3xl xl:text-4xl">
				Welcome to the Authentication Demo
			</h1>
			<p class="text-center text-sm text-muted-foreground mt-6">This is a demo environment. Data is reset periodically.</p>
			<HealthCheck />
		</div>

		{#if browser}
			<NavLinks />
		{/if}
	</div>
</div>
