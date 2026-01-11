<script lang="ts">
	import { defaults, setError, superForm } from 'sveltekit-superforms';
	import { zod4 } from 'sveltekit-superforms/adapters';
	import { TwoFaConfirmFormSchema } from '$lib/schemas/userSchema';
	import { twoFaChallenge } from '$lib/service/authApiService';
	import { toast } from 'svelte-sonner';
	import { userStore } from '$lib/stores';
	import { AuthError } from '$lib/errors/error';
	import { goto } from '$app/navigation';
	import * as Field from '$lib/components/ui/field';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Spinner } from '$lib/components/ui/spinner';

	const { sessionToken } = $props();

	const { form, constraints, errors, enhance, submitting } = superForm(
		defaults(zod4(TwoFaConfirmFormSchema)),
		{
			SPA: true,
			validators: zod4(TwoFaConfirmFormSchema),
			onUpdate: async ({ form }) => {
				if (!form.valid) return;

				const payload = {
					sessionToken: sessionToken as string,
					twoFaCode: form.data.twoFaCode
				};
				try {
					const user = await twoFaChallenge(payload);

					toast.success('Login successful! Redirecting to home page...');

					userStore.login(user);
					setTimeout(() => {
						goto('/');
					}, 0);
				} catch (error) {
					if (error instanceof AuthError && error.status === 400) {
						setError(form, 'twoFaCode', 'Invalid 2FA code');
						return;
					}

					toast.error('Login failed, please try again later.');
				}
			}
		}
	);
</script>

<form method="POST" use:enhance>
	<Field.Set>
		<Field.Group>
			<Field.Field>
				<Input
					id="twoFaCode"
					autocomplete="off"
					name="twoFaCode"
					placeholder="Your 2FA code"
					bind:value={$form.twoFaCode}
					aria-invalid={$errors.twoFaCode ? 'true' : undefined}
					{...$constraints.twoFaCode}
				/>
				{#if $errors.twoFaCode}
					<Field.Error>{$errors.twoFaCode}</Field.Error>
				{/if}
			</Field.Field>
		</Field.Group>
	</Field.Set>

	<Button type="submit" disabled={$submitting} class="mt-6 w-full">
		{#if $submitting}
			<Spinner class="mr-2 h-4 w-4 animate-spin" />
			Submitting...
		{:else}
			Submit
		{/if}
	</Button>
</form>
