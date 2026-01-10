<script lang="ts">
	import { userStore } from '$lib/stores';
	import { goto } from '$app/navigation';
	import { defaults, setError, superForm } from 'sveltekit-superforms';
	import { UpdateUserPasswordFormSchema } from '$lib/schemas/userSchema';
	import { zod4 } from 'sveltekit-superforms/adapters';
	import { updatePassword } from '$lib/service/authApiService';
	import { toast } from 'svelte-sonner';
	import { AuthError } from '$lib/errors/error';
	import * as Field from '$lib/components/ui/field/index.js';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Spinner } from '$lib/components/ui/spinner';

	const { form, constraints, errors, enhance, submitting, reset } = superForm(
		defaults(zod4(UpdateUserPasswordFormSchema)),
		{
			SPA: true,
			validators: zod4(UpdateUserPasswordFormSchema),
			onUpdate: async ({ form }) => {
				if (!form.valid) return;

				const { confirmNewPassword, ...payload } = form.data;
				void confirmNewPassword;

				try {
					const user = await updatePassword(payload);
					console.log(user);

					toast.success('Password updated successfully! Redirecting to home page...');
					userStore.login(user);

					setTimeout(() => {
						goto('/');
					}, 0);
				} catch (error) {
					console.error(error);
					if (error instanceof AuthError && error.status === 401) {
						setError(form, 'oldPassword', 'Invalid password');
						return;
					}

					toast.error('Login failed, please try again later.');
				} finally {
					form.data.oldPassword = '';
					form.data.newPassword = '';
					form.data.confirmNewPassword = '';
				}
			}
		}
	);
</script>

<form method="POST" use:enhance>
	<Field.Set>
		<Field.Legend>Change Password</Field.Legend>
		<Field.Description>Change your password below.</Field.Description>

		<Field.Group>
			<Field.Field>
				<Field.Label for="oldPassword">Old Password</Field.Label>
				<Input
					id="oldPassword"
					autocomplete="off"
					type="password"
					name="oldPassword"
					placeholder="Your old password"
					bind:value={$form.oldPassword}
					aria-invalid={$errors.oldPassword ? 'true' : undefined}
					{...$constraints.oldPassword}
				/>
				{#if $errors.oldPassword}
					<Field.Error>{$errors.oldPassword}</Field.Error>
				{/if}
			</Field.Field>

			<Field.Field>
				<Field.Label for="newPassword">New Password</Field.Label>
				<Input
					id="newPassword"
					autocomplete="off"
					type="password"
					name="newPassword"
					placeholder="Your new password"
					bind:value={$form.newPassword}
					aria-invalid={$errors.newPassword ? 'true' : undefined}
					{...$constraints.newPassword}
				/>
				{#if $errors.newPassword}
					<Field.Error>{$errors.newPassword}</Field.Error>
				{/if}
			</Field.Field>

			<Field.Field>
				<Field.Label for="confirmNewPassword">Confirm New Password</Field.Label>
				<Input
					id="confirmNewPassword"
					autocomplete="off"
					type="password"
					name="confirmNewPassword"
					placeholder="Confirm your new password"
					bind:value={$form.confirmNewPassword}
					aria-invalid={$errors.confirmNewPassword ? 'true' : undefined}
					{...$constraints.confirmNewPassword}
				/>
				{#if $errors.confirmNewPassword}
					<Field.Error>{$errors.confirmNewPassword}</Field.Error>
				{/if}
			</Field.Field>
		</Field.Group>
	</Field.Set>

	<Button type="submit" disabled={$submitting} class="mt-6 w-full">
		{#if $submitting}
			<Spinner class="mr-2 h-4 w-4 animate-spin" />
			Submitting…
		{:else}
			Submit
		{/if}
	</Button>
</form>
