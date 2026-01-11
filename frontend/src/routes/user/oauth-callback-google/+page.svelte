<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { userStore } from '$lib/stores';
	import { getUserProfile } from '$lib/service/authApiService';
	import { toast } from 'svelte-sonner';

	onMount(async () => {
		const token = page.url.searchParams.get('token');

		if (token) {
			try {
				userStore.saveToken(token);
				const user = await getUserProfile();

				userStore.login({ ...user, token });
				toast.success('Successfully logged in with Google OAuth!');
				goto('/', { replaceState: true });
			} catch {
				toast.error('Failed to log in with Google OAuth, please try again.');
				userStore.logout();
				goto('/login', { replaceState: true });
			}
		} else {
			toast.error('Failed to log in with Google OAuth, please try again.');
			goto('/login', { replaceState: true });
		}
	});
</script>
