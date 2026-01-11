<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { healthCheck } from '$lib/service/authApiService';
	import { onMount } from 'svelte';

	let backendHealthCheck = false;

	onMount(async () => {
		backendHealthCheck = await healthCheck();
	});
</script>

<div class="text-sm text-muted-foreground">
	{#if backendHealthCheck}
		<Badge variant="secondary" class="font-semibold text-green-600">Backend is ready</Badge>
	{:else}
		<Badge variant="secondary" class="font-semibold text-red-600">Backend is waking up...</Badge>
	{/if}
</div>
