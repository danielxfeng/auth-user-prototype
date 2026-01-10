<script lang="ts">
	import RegisterForm from './RegisterForm.svelte';
	import { onMount } from 'svelte';
	import { userStore } from '$lib/stores';
	import { goto } from '$app/navigation';

  onMount(() => {
    if ($userStore.user) {
      goto('/user/profile', { replaceState: true });
    }
  });
  
</script>

<div class="px-6">
	<RegisterForm />
</div>
