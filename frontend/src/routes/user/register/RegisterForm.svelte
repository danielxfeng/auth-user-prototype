<script lang="ts">
	import { superForm, defaults, setError } from 'sveltekit-superforms';
	import { zod4 } from 'sveltekit-superforms/adapters';
	import { CreateUserFormSchema } from '$lib/schemas/userSchema';
	import { registerUser } from '$lib/service/authApiService';
	import { toast } from 'svelte-sonner';
	import { goto } from '$app/navigation';
	import { AuthError } from '$lib/errors/error';
	import * as Field from '$lib/components/ui/field/index.js';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Spinner } from '$lib/components/ui/spinner';

	const { form, constraints, errors, enhance, submitting } = superForm(
		defaults(zod4(CreateUserFormSchema)),
		{
			SPA: true,
			validators: zod4(CreateUserFormSchema),
			onUpdate: async ({ form }) => {
				if (!form.valid) return;

				try {
					const { confirmPassword, ...payload } = form.data;
					void confirmPassword;

					await registerUser(payload);
					toast.success('Registration successful! Redirecting to login...');

					setTimeout(() => {
						goto('/users/login/');
					}, 2000);
				} catch (error) {
					if (error instanceof AuthError && error.status === 409) {
						setError(form, 'username', 'Username or Email already taken');
						setError(form, 'email', 'Username or Email already taken');
						return;
					}

					toast.error('Registration failed, please try again later.');
				}
			}
		}
	);
</script>


	<form method="POST" use:enhance>
		<Field.Set>
			<Field.Legend>Register</Field.Legend>
			<Field.Description
				>Create a new account by filling out the information below.</Field.Description
			>

			<Field.Group>
				<Field.Field>
					<Field.Label for="username">Username</Field.Label>
					<Input
						id="username"
						autocomplete="off"
						name="username"
						placeholder="Your username"
						bind:value={$form.username}
						aria-invalid={$errors.username ? 'true' : undefined}
						{...$constraints.username}
					/>
					{#if $errors.username}
						<Field.Error>{$errors.username}</Field.Error>
					{/if}
				</Field.Field>

				<Field.Field>
					<Field.Label for="email">Email</Field.Label>
					<Input
						id="email"
						type="email"
						autocomplete="off"
						name="email"
						placeholder="Your email"
						bind:value={$form.email}
						aria-invalid={$errors.email ? 'true' : undefined}
						{...$constraints.email}
					/>
					{#if $errors.email}
						<Field.Error>{$errors.email}</Field.Error>
					{/if}
				</Field.Field>

				<Field.Field>
					<Field.Label for="password">Password</Field.Label>
					<Input
						id="password"
						type="password"
						autocomplete="off"
						name="password"
						placeholder="Your password"
						bind:value={$form.password}
						aria-invalid={$errors.password ? 'true' : undefined}
						{...$constraints.password}
					/>
					{#if $errors.password}
						<Field.Error>{$errors.password}</Field.Error>
					{/if}
				</Field.Field>

				<Field.Field>
					<Field.Label for="confirmPassword">Confirm Password</Field.Label>
					<Input
						id="confirmPassword"
						type="password"
						autocomplete="off"
						name="confirmPassword"
						placeholder="Confirm your password"
						bind:value={$form.confirmPassword}
						aria-invalid={$errors.confirmPassword ? 'true' : undefined}
						{...$constraints.confirmPassword}
					/>
					{#if $errors.confirmPassword}
						<Field.Error>{$errors.confirmPassword}</Field.Error>
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
