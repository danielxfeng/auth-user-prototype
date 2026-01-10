<script lang="ts">
	import { get } from 'svelte/store';
	import { userStore } from '$lib/stores';
	import * as Avatar from '$lib/components/ui/avatar/index.js';
	import { Button } from '$lib/components/ui/button';
	import { cn } from '$lib/utils';
	import { Pencil } from 'lucide-svelte';
	import { defaults, setError, superForm } from 'sveltekit-superforms';
	import { zod4 } from 'sveltekit-superforms/adapters';
	import { UpdateUserAvatarFormSchema } from '$lib/schemas/userSchema';
	import { toast } from 'svelte-sonner';
	import { AuthError } from '$lib/errors/error';
	import { updateProfile } from '$lib/service/authApiService';
	import * as Field from '$lib/components/ui/field';
	import { Input } from '$lib/components/ui/input';
	import { Spinner } from '$lib/components/ui/spinner';
	import type { UpdateUserRequest } from '$lib/schemas/types';

	let editing = false;

	const { form, constraints, errors, enhance, submitting } = superForm(
		defaults(zod4(UpdateUserAvatarFormSchema)),
		{
			SPA: true,
			validators: zod4(UpdateUserAvatarFormSchema),
			onUpdate: async ({ form }) => {
				if (!form.valid) return;

				const avatarValue = form.data.avatar ? form.data.avatar : null;
				const { username, email, ...rest } = get(userStore).user!;
				void rest;

				const payload: UpdateUserRequest = {
					username,
					email,
					avatar: avatarValue
				};

				try {
					const updatedUser = await updateProfile(payload);

					toast.success('Avatar updated successfully!');

					userStore.updateUser(updatedUser);

					editing = false;
				} catch (error) {
					if (error instanceof AuthError && error.status === 400) {
						setError(form, 'avatar', 'Invalid avatar URL');
						return;
					}

					toast.error('Avatar update failed, please try again later.');
				}
			}
		}
	);
</script>

{#if $userStore.user}
	<h2 class="flex items-center gap-4 text-3xl font-semibold">
		<div class="group relative">
			<Avatar.Root class="h-16 w-16">
				<Avatar.Image src={$userStore.user.avatar ?? undefined} alt={$userStore.user.username} />
				<Avatar.Fallback class="bg-primary text-background"
					>{$userStore.user.username.charAt(0).toUpperCase()}</Avatar.Fallback
				>
			</Avatar.Root>

			<Button
				variant="ghost"
				class={cn(
					'absolute inset-0 flex h-16 w-16 items-center justify-center rounded-full bg-foreground opacity-0 transition-opacity hover:opacity-70',
					editing && 'pointer-events-none'
				)}
				onclick={() => (editing = true)}
				disabled={editing}
				aria-label="Edit avatar"
			>
				<Pencil class="h-8 w-8 text-foreground" />
			</Button>
		</div>
		{$userStore.user.username}
	</h2>

	{#if editing}
		<form method="POST" use:enhance class="mt-6 space-y-4">
			<Field.Set>
				<Field.Field>
					<Input
						id="avatar"
						type="url"
						autocomplete="off"
						name="avatar"
						placeholder="Your avatar URL"
						bind:value={$form.avatar}
						aria-invalid={$errors.avatar ? 'true' : undefined}
						{...$constraints.avatar}
					/>
					{#if $errors.avatar}
						<Field.Error>{$errors.avatar}</Field.Error>
					{/if}
				</Field.Field>
			</Field.Set>

			<div class="flex w-full gap-4">
				<Button type="submit" disabled={$submitting} class="flex-1">
					{#if $submitting}
						<Spinner class="mr-2 h-4 w-4 animate-spin" />
						Updating...
					{:else}
						Update
					{/if}
				</Button>

				<Button variant="ghost" onclick={() => (editing = false)} type="button" class="flex-1">
					Cancel
				</Button>
			</div>
		</form>
	{/if}
{/if}
