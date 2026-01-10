<script lang="ts">
	import { page } from '$app/state';
	import { userStore } from '$lib/stores';
	import { cn } from '$lib/utils';

	const loggedLinks = [
		{ href: '/profile/', label: 'Profile' },
		{ href: '/settings/', label: 'Settings' }
	];

	const guestLinks = [
		{ href: '/login/', label: 'Login' },
		{ href: '/register/', label: 'Register' }
	];

	$: links = $userStore.user ? loggedLinks : guestLinks;

	const isActive = (href: string) =>
		page.url.pathname === href || page.url.pathname.startsWith(href);
</script>

<div class="flex w-full items-center justify-center">
	{#each links as link (link.href)}
		<a
			href={link.href}
			class={cn(
				'px-4 py-2 text-sm font-medium first:rounded-tl-lg last:rounded-tr-lg',
				isActive(link.href) && 'bg-accent text-primary'
			)}
		>
			{link.label}
		</a>
	{/each}
</div>
