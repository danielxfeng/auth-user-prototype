<script lang="ts">
	import { goto } from "$app/navigation";
	import { Button } from "$lib/components/ui/button";
	import Spinner from "$lib/components/ui/spinner/spinner.svelte";
	import { logoutUser } from "$lib/service/authApiService";
	import { userStore } from "$lib/stores";
	import { toast } from "svelte-sonner";

  let logoutInProgress = false;

  const logoutHandler = async () => {
    logoutInProgress = true;
    try {
      await logoutUser();
      
      toast.success("Logged out successfully, redirecting to home page...");
    } catch {
      toast.warning("Failed to log out on server, try to log out locally, redirecting to home page...");
    } finally {
      userStore.logout();
      logoutInProgress = false;
      goto("/");
    }
  };
  
</script>

<Button variant="outline" onclick={logoutHandler} disabled={logoutInProgress}>
  {#if logoutInProgress}
    <Spinner class="mr-2 h-4 w-4 animate-spin"/> Logging out...
  {:else}
    Logout
  {/if}
</Button>