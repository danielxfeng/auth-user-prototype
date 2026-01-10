<script lang="ts">
  import { onMount } from 'svelte';
  import { userStore } from '$lib/stores';
  import { goto } from '$app/navigation';

  onMount(() => {
    userStore.logout();
    goto('/user/login', { replaceState: true });
  });
</script>
