<script lang="ts">
	import LoginForm from './LoginForm.svelte';
	import { onMount } from 'svelte';
	import { userStore } from '$lib/stores';
	import { goto } from '$app/navigation';
	import TwoFaForm from './TwoFaForm.svelte';
	import { fly } from 'svelte/transition';

	let status: 'login' | '2fa' = 'login';
	let sessionToken: string = '';

	const goto2fa: (session: string) => void = (session) => {
		sessionToken = session;
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
		<div class="w-full" out:fly={{ y: -20, duration: 500 }}>
			<LoginForm {goto2fa} />
		</div>
	{/if}
	{#if status === '2fa'}
		<div class="w-full" in:fly={{ y: -20, delay: 500, duration: 500 }}>
			<TwoFaForm {sessionToken} />
		</div>
	{/if}
</div>
