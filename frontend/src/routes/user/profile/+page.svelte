<script lang="ts">
	import { userStore } from '$lib/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import ProfileTitle from './ProfileTitle.svelte';
	import Friends from './Friends.svelte';

	onMount(() => {
		if (!$userStore.user) {
			goto('/user/login', { replaceState: true });
		}
	});
</script>

{#if $userStore.user}
	<div class="flex flex-col gap-8 px-6">
		<ProfileTitle />
		<Friends />
	</div>
{/if}
