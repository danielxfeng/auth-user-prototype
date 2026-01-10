<script lang="ts">
	import { page } from '$app/state';
	import { userStore } from '$lib/stores';
	import * as ButtonGroup from '$lib/components/ui/button-group/index.js';
	import { Button } from '$lib/components/ui/button/index.js';

	const loggedLinks = [
		{ href: '/profile/', label: 'Profile' },
		{ href: '/settings/', label: 'Settings' },
		{ href: '/friends/', label: 'Friends' }
	];

	const guestLinks = [
		{ href: '/login/', label: 'Login' },
		{ href: '/register/', label: 'Register' }
	];

	$: links = $userStore.user ? loggedLinks : guestLinks;

	const isActive = (href: string) =>
		page.url.pathname === href || page.url.pathname.startsWith(href);
</script>

<ButtonGroup.Root class="flex w-full overflow-hidden">
	{#each links as link (link.href)}
		<Button href={link.href} variant={isActive(link.href) ? 'default' : 'outline'}>
			{link.label}
		</Button>
	{/each}
</ButtonGroup.Root>
