<script lang="ts">
	import { userStore } from '$lib/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import ProfileTitle from './ProfileTitle.svelte';

	onMount(() => {
		if (!$userStore.user) {
			goto('/user/login', { replaceState: true });
		}
	});
</script>

{#if $userStore.user}
	<div class="px-6">
		<ProfileTitle />
		<p></p>
	</div>
{/if}
