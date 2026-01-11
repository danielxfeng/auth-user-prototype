<script lang="ts">
	import * as Avatar from '$lib/components/ui/avatar';
	import { Button } from '$lib/components/ui/button';
	import * as Field from '$lib/components/ui/field';
	import { Input } from '$lib/components/ui/input';
	import { Spinner } from '$lib/components/ui/spinner';
	import { AuthError } from '$lib/errors/error';
	import type { GetFriendsResponse, UsersResponse } from '$lib/schemas/types';
	import { AddNewFriendFormSchema } from '$lib/schemas/userSchema';
	import { addFriend, getAllUsers, getFriends } from '$lib/service/authApiService';
	import { cn } from '$lib/utils';
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { defaults, setError, superForm } from 'sveltekit-superforms';
	import { zod4 } from 'sveltekit-superforms/adapters';
	import { userStore } from '$lib/stores';
	import { logger } from '$lib/config/logger';

	let friends: GetFriendsResponse = [];
	let users: UsersResponse = [];

	onMount(async () => {
		try {
			[users, friends] = await Promise.all([getAllUsers(), getFriends()]);
		} catch (error) {
			logger.error('Failed to load friends list:', error);
			toast.error('Failed to load friends list');
		}
	});

	const { form, constraints, errors, enhance, submitting, reset, submit } = superForm(
		defaults(zod4(AddNewFriendFormSchema)),
		{
			SPA: true,
			validators: zod4(AddNewFriendFormSchema),
			onUpdate: async ({ form }) => {
				if (!form.valid) return;

				const userId = users.find((u) => u.username === form.data.username)?.id;
				if (!userId) {
					setError(form, 'username', 'User not found');
					return;
				}

				try {
					const payload = { userId };
					reset();

					await addFriend(payload);
					toast.success('Friend added successfully!');

					friends = await getFriends();
				} catch (error) {
					if (error instanceof AuthError && (error.status === 400 || error.status === 404)) {
						setError(form, 'username', 'Invalid username');
						return;
					}

					logger.error('Failed to add friend:', error);
					toast.error('Failed to add friend, please try again later.');
				}
			}
		}
	);

	let focused = false;
	$: query = ($form.username ?? '').trim().toLowerCase();
	$: suggestions =
		query.length === 0
			? []
			: users.filter(
					(u) =>
						u.username.toLowerCase().includes(query) &&
						$userStore.user?.username !== u.username &&
						!friends.some((f) => f.username === u.username)
				);
</script>

<form method="POST" use:enhance class="space-y-4">
	<Field.Set>
		<Field.Field class="relative">
			<Input
				id="avatar"
				autocomplete="off"
				name="username"
				placeholder="Your friend's username"
				bind:value={$form.username}
				aria-invalid={$errors.username ? 'true' : undefined}
				onfocus={() => (focused = true)}
				onblur={() => (focused = false)}
				{...$constraints.username}
			/>
			{#if suggestions.length > 0 && focused}
				<div
					class="absolute top-full z-10 mt-1 w-full rounded-lg border border-border bg-popover px-4 py-2 shadow-lg"
				>
					{#each suggestions as suggestion (suggestion.username)}
						<button
							type="button"
							class="w-full rounded-md px-3 py-2 text-left text-sm hover:bg-accent focus:bg-accent"
							on:mousedown|preventDefault={() => {
								$form.username = suggestion.username;
								submit();
							}}
						>
							{suggestion.username}
						</button>
					{/each}
				</div>
			{/if}

			{#if $errors.username}
				<Field.Error>{$errors.username}</Field.Error>
			{/if}
		</Field.Field>
	</Field.Set>

	<Button type="submit" disabled={$submitting} class="w-full">
		{#if $submitting}
			<Spinner class="mr-2 h-4 w-4 animate-spin" />
			Adding...
		{:else}
			Add Friend
		{/if}
	</Button>
</form>

{#if friends.length > 0}
	<h2 class="text-2xl font-semibold">Friends</h2>
	<ul class="flex w-full max-w-2xl flex-col gap-3 rounded-xl border border-border p-4">
		{#each friends as friend (friend.username)}
			<li class="grid items-center gap-4" style="grid-template-columns: 40px minmax(0, 1fr) 20px;">
				<Avatar.Root class="h-10 w-10">
					<Avatar.Image src={friend.avatar ?? undefined} alt={friend.username} />
					<Avatar.Fallback>
						{friend.username.charAt(0).toUpperCase()}
					</Avatar.Fallback>
				</Avatar.Root>

				<span class="min-w-0 truncate font-medium">
					{friend.username}
				</span>

				<span
					title={friend.online ? 'Online' : 'Offline'}
					class={cn('h-6 w-6 rounded-full', friend.online ? 'bg-green-500/20' : 'bg-gray-500/20')}
				></span>
			</li>
		{/each}
	</ul>
{/if}
