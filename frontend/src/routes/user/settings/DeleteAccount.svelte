<script lang="ts">
	import { userStore } from '$lib/stores';
	import { goto } from '$app/navigation';
	import { deleteAccount } from '$lib/service/authApiService';
	import { toast } from 'svelte-sonner';
	import { buttonVariants } from '$lib/components/ui/button/button.svelte';
	import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';
	import { Spinner } from '$lib/components/ui/spinner';

	let deleting = false;

	const deleteAccountHandler = async () => {
		deleting = true;
		try {
			await deleteAccount();
			userStore.logout();
			toast.success('Account deleted successfully, navigating to home page...');
			
      goto('/');
		} catch {
			toast.error('Failed to delete account. Please try again.');
		} finally {
			deleting = false;
		}
	};
</script>

<AlertDialog.Root>
	<AlertDialog.Trigger class={buttonVariants({ variant: 'destructive' })}>
		{#if deleting}
			<Spinner class="mr-2 h-4 w-4 animate-spin"/> Deleting...
		{:else}
			Delete Account
		{/if}
	</AlertDialog.Trigger>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>Are you absolutely sure?</AlertDialog.Title>
			<AlertDialog.Description>
				This action cannot be undone. This will permanently delete your account and remove your data
				from our servers.
			</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
			<AlertDialog.Action onclick={deleteAccountHandler} disabled={deleting}>Continue</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
