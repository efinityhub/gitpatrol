<script lang="ts">
  import { authStore } from '$lib/auth.svelte';
  import { reposStore } from '$lib/repos.svelte';
  import { incidentsStore } from '$lib/incidents.svelte';
  import { healthStore } from '$lib/health.svelte';
  import { fade } from 'svelte/transition';

  let username = $state('');
  let password = $state('');
  let loading = $state(false);

  async function handleSubmit() {
    loading = true;
    const success = await authStore.handleAuth(username, password);
    if (success && authStore.isAuthenticated) {
      username = '';
      password = '';
      reposStore.fetchRepos();
      incidentsStore.fetchIncidents();
      healthStore.fetchHealth();
      healthStore.setupHealthPolling();
    }
    loading = false;
  }
</script>

<div class="modal-overlay" transition:fade>
  <div class="modal-content" style="max-width: 400px; padding: 40px;">
    <div style="text-align: center; margin-bottom: 40px;">
      <h1 style="font-size: 2.5rem; margin-bottom: 8px;">GitPatrol ⚡</h1>
      <h2>{authStore.needsBootstrap ? 'System Bootstrap' : 'Secure Login'}</h2>
      <p style="color: var(--efinity-text-muted);">{authStore.needsBootstrap ? 'Create the initial admin account to secure GitPatrol.' : 'Authenticate to access GitPatrol.'}</p>
    </div>
    
    <label for='username'>USERNAME</label>
    <input id='username' type="text" bind:value={username} placeholder="username" />
    
    <label for='password'>PASSWORD</label>
    <input id='password' type="password" bind:value={password} placeholder="" onkeydown={(e) => e.key === 'Enter' && handleSubmit()} />
    
    <button style="width: 100%; margin-top: 32px;" onclick={handleSubmit} disabled={loading}>
      {loading ? 'PROCESSING...' : authStore.needsBootstrap ? 'BOOTSTRAP SYSTEM' : 'ACCESS DASHBOARD'}
    </button>

    {#if !authStore.needsBootstrap}
      <p style="text-align: center; font-size: 0.75rem; color: var(--efinity-text-muted); margin-top: 24px;">
        Contact your administrator for access.
      </p>
    {/if}
  </div>
</div>
