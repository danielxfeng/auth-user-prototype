<script lang="ts">
	import { userStore } from '$lib/stores';
	import { defaults, setError, superForm } from 'sveltekit-superforms';
	import { zod4 } from 'sveltekit-superforms/adapters';
	import { TwoFaDisableRequestSchema } from '$lib/schemas/userSchema';
	import { disableTwoFa } from '$lib/service/authApiService';
	import { toast } from 'svelte-sonner';
	import { AuthError } from '$lib/errors/error';
	import * as Field from '$lib/components/ui/field';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Spinner } from '$lib/components/ui/spinner';
	import { logger } from '$lib/config/logger';

	const { closeShowTwoFaForm } = $props();

	const { form, constraints, errors, enhance, submitting } = superForm(
		defaults(zod4(TwoFaDisableRequestSchema)),
		{
			SPA: true,
			validators: zod4(TwoFaDisableRequestSchema),
			onUpdate: async ({ form }) => {
				if (!form.valid) return;

				try {
					await disableTwoFa(form.data);

					toast.success('2FA disabled successfully, please log in again.');

					userStore.logout();
					closeShowTwoFaForm();
					setTimeout(() => {
						goto('/login');
					}, 0);
				} catch (error) {
					if (error instanceof AuthError && error.status === 401) {
						setError(form, 'password', 'Invalid password');
						return;
					}

					logger.error('Disabling 2FA error:', error);
					toast.error('Disabling 2FA failed, please try again later.');
				} finally {
					form.data.password = '';
				}
			}
		}
	);
</script>

{#if $userStore.user?.twoFa}
	<form method="POST" use:enhance>
		<Field.Set>
			<Field.Group>
				<Field.Field>
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
				Disabling...
			{:else}
				Disable
			{/if}
		</Button>
	</form>
{/if}
