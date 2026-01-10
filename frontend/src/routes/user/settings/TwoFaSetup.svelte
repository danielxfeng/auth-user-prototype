<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { userStore } from '$lib/stores';
	import DisableTwoFa from './DisableTwoFa.svelte';

	$: twoFaEnabled = $userStore.user?.twoFa ?? false;
	let showTwoFaForm = false;
</script>

<Button
	variant={twoFaEnabled ? 'destructive' : 'secondary'}
	onclick={() => {
		if (!showTwoFaForm) showTwoFaForm = true;
	}}
	disabled={$userStore.user?.googleOauthId ? true : false}
	class="w-full"
>
	{#if twoFaEnabled}
		Disable 2FA
	{:else}
		Enable 2FA
	{/if}
</Button>
{#if showTwoFaForm}
	{#if twoFaEnabled}
		<DisableTwoFa closeShowTwoFaForm={() => (showTwoFaForm = false)} />
	{:else}
		<p>2FA enabling form goes here.</p>
	{/if}
{/if}
