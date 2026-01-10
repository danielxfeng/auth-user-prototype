<script lang="ts">
	import LoginForm from './LoginForm.svelte';

	let status: 'login' | '2fa' = 'login';

	const goto2fa: () => void = () => {
		status = '2fa';
	};
</script>

<div class="px-6">
	{#if status === 'login'}
		<LoginForm {goto2fa} />
	{:else if status === '2fa'}
		<p>Two-Factor Authentication Form Placeholder</p>
	{/if}
</div>
