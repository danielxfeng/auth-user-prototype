<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { userStore } from '$lib/stores';
	import { getUserProfile } from '$lib/service/authApiService';
	import { toast } from 'svelte-sonner';
	import { logger } from '$lib/config/logger';

	onMount(async () => {
		const token = page.url.searchParams.get('token');

		if (token) {
			try {
				userStore.saveToken(token);
				const user = await getUserProfile();

				userStore.login({ ...user, token });
				toast.success('Successfully logged in with Google OAuth!');
				goto('/', { replaceState: true });
			} catch (error) {
				toast.error('Failed to log in with Google OAuth, please try again.');
				logger.error('OAuth login error:', error);
				userStore.logout();
				goto('/user/login', { replaceState: true });
			}
		} else {
			toast.error('Failed to log in with Google OAuth, please try again.');
			logger.error('OAuth callback missing token parameter');
			goto('/user/login', { replaceState: true });
		}
	});
</script>
