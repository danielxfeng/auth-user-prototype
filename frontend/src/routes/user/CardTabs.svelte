<script lang="ts">
	import { page } from '$app/state';
	import { userStore } from '$lib/stores';
	import * as ButtonGroup from '$lib/components/ui/button-group/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { cn } from '$lib/utils';

	const loggedLinks = [
		{ href: '/user/profile/', label: 'Profile' },
		{ href: '/user/settings/', label: 'Settings' }
	];

	const guestLinks = [
		{ href: '/user/login/', label: 'Login' },
		{ href: '/user/register/', label: 'Register' }
	];

	$: links = $userStore.user ? loggedLinks : guestLinks;

	const isActive = (href: string) => {
		const normalizedHref = href.endsWith('/') ? href.slice(0, -1) : href;
		return page.url.pathname === normalizedHref || page.url.pathname.startsWith(normalizedHref);
	};
</script>

<ButtonGroup.Root class="flex w-full overflow-hidden">
	{#each links as link (link.href)}
		<Button
			href={link.href}
			variant="outline"
			class={cn(
				'flex-1',
				isActive(link.href) &&
					'bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground'
			)}
		>
			{link.label}
		</Button>
	{/each}
</ButtonGroup.Root>
