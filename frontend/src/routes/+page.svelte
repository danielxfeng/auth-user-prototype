<script>
	import { userStore } from '$lib/stores';
	import { fade, fly } from 'svelte/transition';
	import { onMount } from 'svelte';

	let showTitle = false;

	onMount(() => {
		showTitle = true;
	});
</script>

<div class="flex flex-1 flex-col items-center justify-center gap-10">
	<div class="flex max-w-4xl flex-col items-center justify-center gap-12 p-4">
		{#if showTitle}
			<h1
				class="text-center text-2xl font-semibold text-primary lg:text-3xl"
				in:fly={{ y: 20, duration: 400, delay: 400 }}
			>
				Welcome to the Authentication Demo
			</h1>

			<div class="flex items-center justify-center" in:fade={{ duration: 500, delay: 600 }}>
				{#if $userStore.user}
					<a href="/profile/" role="button" class="btn"> Go to Profile </a>
				{:else}
					<div class="flex gap-4">
						<a href="/login/" role="button" class="btn"> Login </a>
						<a href="/register/" role="button" class="btn"> Register </a>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</div>
