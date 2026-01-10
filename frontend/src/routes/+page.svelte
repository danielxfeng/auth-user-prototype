<script>
	import { browser } from '$app/environment';
	import NavLinks from './NavLinks.svelte';
</script>

<div class="flex flex-1 flex-col items-center justify-center gap-10">
	<div class="flex max-w-4xl flex-col items-center justify-center gap-18 p-4">
		<h1 class="animate-enter text-center text-2xl font-semibold lg:text-3xl xl:text-4xl">
			Welcome to the Authentication Demo
		</h1>

		{#if browser}
      <NavLinks />
    {/if}
	</div>
</div>
