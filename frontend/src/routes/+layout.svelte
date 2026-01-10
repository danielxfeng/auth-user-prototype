<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import Toast from '$lib/components/Toast.svelte';

	let { children } = $props();
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>
<div class="relative flex min-h-screen flex-col items-center justify-center">
	{@render children()}
	<Toast />
</div>
