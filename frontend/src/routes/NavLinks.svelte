<script lang="ts">
	import { userStore } from '$lib/stores';
	import { Button } from '$lib/components/ui/button/index.js';
</script>

<div class="flex animate-enter-soft items-center justify-center">
	{#if $userStore.user}
		<Button href="/user/profile/" variant="default">Go to Profile</Button>
	{:else}
		<div class="flex gap-4">
			<Button href="/user/login/" variant="default">Login</Button>
			<Button href="/user/register/" variant="outline">Register</Button>
		</div>
	{/if}
</div>
