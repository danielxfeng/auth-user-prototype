<script>
	import { userStore } from '$lib/stores';
	import { browser } from '$app/environment';
</script>

<div class="flex flex-1 flex-col items-center justify-center gap-10">
	<div class="flex max-w-4xl flex-col items-center justify-center gap-12 p-4">
		<h1 class="animate-enter text-center text-2xl font-semibold lg:text-3xl xl:text-4xl">
			Welcome to the Authentication Demo
		</h1>

		{#if browser}
			<div class="flex animate-enter-soft items-center justify-center">
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
