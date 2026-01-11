<script lang="ts">
	import { userStore } from '$lib/stores';
	import { onMount } from 'svelte';
	import QRCode from 'qrcode';
	import { defaults, setError, superForm } from 'sveltekit-superforms';
	import { zod4 } from 'sveltekit-superforms/adapters';
	import { TwoFaConfirmFormSchema } from '$lib/schemas/userSchema';
	import type { TwoFaConfirmRequest } from '$lib/schemas/types';
	import { twoFaConfirm } from '$lib/service/authApiService';
	import { AuthError } from '$lib/errors/error';
	import { toast } from 'svelte-sonner';
	import * as Field from '$lib/components/ui/field';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Spinner } from '$lib/components/ui/spinner';

	const { twoFaSetupData, closeShowTwoFaForm } = $props();

	let canvas: HTMLCanvasElement;

	onMount(() => {
		QRCode.toCanvas(canvas, twoFaSetupData.twoFaUri, {
			width: 200
		});
	});

	const { form, constraints, errors, enhance, submitting } = superForm(
		defaults(zod4(TwoFaConfirmFormSchema)),
		{
			SPA: true,
			validators: zod4(TwoFaConfirmFormSchema),
			onUpdate: async ({ form }) => {
				if (!form.valid) return;

				const payload: TwoFaConfirmRequest = {
					twoFaCode: form.data.twoFaCode,
					setupToken: twoFaSetupData.setupToken
				};

				try {
					const user = await twoFaConfirm(payload);

					userStore.login(user);

					toast.success('2FA enabled successfully!');
					closeShowTwoFaForm();
				} catch (error) {
					if (error instanceof AuthError && error.status === 400) {
						setError(form, 'twoFaCode', 'Invalid 2FA code');
						return;
					}

					toast.error('Enabling 2FA failed, please try again later.');
				} finally {
					form.data.twoFaCode = '';
				}
			}
		}
	);
</script>

<div class="flex flex-col items-center justify-between gap-4 p-4 lg:flex-row">
	<canvas bind:this={canvas} class="shrink-0"></canvas>

	<div class="w-full flex-1 lg:w-1/2">
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
					Confirming...
				{:else}
					Confirm
				{/if}
			</Button>

			<Button type="button" variant="ghost" class="mt-4 w-full" onclick={closeShowTwoFaForm}>
				Cancel
			</Button>
		</form>
	</div>
</div>
