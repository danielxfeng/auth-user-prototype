<script lang="ts">
	import { userStore } from '$lib/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import DeleteAccount from './DeleteAccount.svelte';
	import Logout from './Logout.svelte';
	import ChangePasswordForm from './ChangePasswordForm.svelte';
	import TwoFaSetup from './TwoFaSetup.svelte';

	onMount(() => {
		if (!$userStore.user) {
			goto('/user/login', { replaceState: true });
		}
	});
</script>

{#if $userStore.user}
	<div class="flex flex-col gap-8 px-6 flex-1">
		<TwoFaSetup />
		<hr />
		<ChangePasswordForm />
		<hr />
		<Logout />
		<DeleteAccount />
	</div>
{/if}
