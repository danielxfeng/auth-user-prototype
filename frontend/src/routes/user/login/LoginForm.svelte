<script lang="ts">
	import { superForm, defaults, setError } from 'sveltekit-superforms';
	import { zod4 } from 'sveltekit-superforms/adapters';
	import { LoginUserByIdentifierRequestSchema } from '$lib/schemas/userSchema';
	import { loginUser } from '$lib/service/authApiService';
	import { toast } from 'svelte-sonner';
	import { goto } from '$app/navigation';
	import { AuthError } from '$lib/errors/error';
	import * as Field from '$lib/components/ui/field/index.js';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Spinner } from '$lib/components/ui/spinner';
	import { cfg } from '$lib/config/config';
	import { userStore } from '$lib/stores';
	import type { UserWithTokenResponse } from '$lib/schemas/types';

	let { goto2fa } = $props();

	const { form, constraints, errors, enhance, submitting } = superForm(
		defaults(zod4(LoginUserByIdentifierRequestSchema)),
		{
			SPA: true,
			validators: zod4(LoginUserByIdentifierRequestSchema),
			onUpdate: async ({ form }) => {
				if (!form.valid) return;

				try {
					const user = await loginUser(form.data);
					if ('message' in user && user.message === '2FA_REQUIRED') {
						toast.info('Please enter your 2FA code to continue.');
						goto2fa(user.sessionToken);
						return;
					}

					toast.success('Login successful! Redirecting to home page...');

					userStore.login(user as UserWithTokenResponse);

					setTimeout(() => {
						goto('/');
					}, 0);
				} catch (error) {
					if (error instanceof AuthError && error.status === 401) {
						setError(form, 'identifier', 'Invalid username or email');
						setError(form, 'password', 'Invalid username or email');
						return;
					}

					toast.error('Login failed, please try again later.');
				} finally {
					form.data.password = '';
				}
			}
		}
	);
</script>

<form method="POST" use:enhance>
	<Field.Set>
		<Field.Legend>Login</Field.Legend>
		<Field.Description>Log in to your account by entering your credentials below.</Field.Description
		>

		<Button href={`${cfg.apiBaseUrl}/google/login`} variant="default">Login with Google</Button>

		<Field.Group>
			<Field.Field>
				<Field.Label for="identifier">Username or Email</Field.Label>
				<Input
					id="identifier"
					autocomplete="off"
					name="identifier"
					placeholder="Your username or email"
					bind:value={$form.identifier}
					aria-invalid={$errors.identifier ? 'true' : undefined}
					{...$constraints.identifier}
				/>
				{#if $errors.identifier}
					<Field.Error>{$errors.identifier}</Field.Error>
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
		</Field.Group>
	</Field.Set>

	<Button type="submit" disabled={$submitting} class="mt-6 w-full">
		{#if $submitting}
			<Spinner class="mr-2 h-4 w-4 animate-spin" />
			Logging in...
		{:else}
			Login
		{/if}
	</Button>
</form>
