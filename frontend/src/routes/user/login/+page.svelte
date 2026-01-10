<script lang="ts">
	import LoginForm from './LoginForm.svelte';
	import { onMount } from 'svelte';
	import { userStore } from '$lib/stores';
	import { goto } from '$app/navigation';

	let status: 'login' | '2fa' = 'login';

	const goto2fa: () => void = () => {
		status = '2fa';
	};

	onMount(() => {
		if ($userStore.user) {
			goto('/user/profile', { replaceState: true });
		}
	});
</script>

<div class="px-6">
	{#if status === 'login'}
		<LoginForm {goto2fa} />
	{:else if status === '2fa'}
		<p>Two-Factor Authentication Form Placeholder</p>
	{/if}
</div>
