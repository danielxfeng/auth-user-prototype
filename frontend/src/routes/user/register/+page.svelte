<script lang="ts">
  import { browser } from '$app/environment';
	import RegisterForm from './RegisterForm.svelte';
</script>

<div class="px-6">
  {#if browser}
    <RegisterForm />
  {/if}
</div>