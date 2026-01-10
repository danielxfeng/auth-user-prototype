<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { startTwoFaSetup } from '$lib/service/authApiService';
	import { userStore } from '$lib/stores';
	import { toast } from 'svelte-sonner';
	import DisableTwoFa from './DisableTwoFa.svelte';
	import type { TwoFaSetupResponse } from '$lib/schemas/types';
	import TwoFaConfirmForm from './TwoFaConfirmForm.svelte';

	$: twoFaEnabled = $userStore.user?.twoFa ?? false;
	let showTwoFaForm = false;
	let twoFaSetupData: TwoFaSetupResponse | null = null;

	const twoFaHandler = async () => {
		try {
			twoFaSetupData = await startTwoFaSetup();
		} catch {
			toast.error('Failed to start 2FA setup, please try again later.');
		}
	};
</script>

<Button
	variant={twoFaEnabled ? 'destructive' : 'default'}
	onclick={async () => {
		if (!showTwoFaForm) {
			showTwoFaForm = true;
			if (!twoFaEnabled) {
				await twoFaHandler();
			}
		}
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
	{:else if twoFaSetupData}
		<TwoFaConfirmForm closeShowTwoFaForm={() => (showTwoFaForm = false)} {twoFaSetupData} />
	{/if}
{/if}
