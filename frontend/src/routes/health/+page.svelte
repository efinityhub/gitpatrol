<script lang="ts">
  import { fade } from 'svelte/transition';
  import { healthStore } from '$lib/health.svelte';
  import { incidentsStore } from '$lib/incidents.svelte';
  import { apiFetch } from '$lib/client';
  import Icon from '$lib/components/Icon.svelte';
  import { formatBytes } from '$lib/utils';

  async function clearIncident(id: number) {
    await apiFetch(`/api/incidents/${id}/resolve`, { method: 'POST' });
    incidentsStore.fetchIncidents();
  }
</script>

<div class="page-container" transition:fade>
  <div class="header">
    <h1 class="display-lg">System Health</h1>
    
    {#if healthStore.healthStatus}
      <div class="status-chip status-{healthStore.healthStatus.status}">
        <span class="dot"></span>
        {healthStore.healthStatus.status.toUpperCase()}
      </div>
    {:else}
      <div class="status-chip status-offline">
        <span class="dot"></span>
        CHECKING...
      </div>
    {/if}
  </div>

  <div class="metrics-grid">
    {#if healthStore.healthStatus}
      <!-- Network / Internet -->
      <div class="metric-card">
        <h3 class="metric-title">NETWORK LINK</h3>
        <div class="metric-value" style="color: {healthStore.healthStatus.checks.internet.connected ? 'var(--primary)' : 'var(--error)'};">
          {healthStore.healthStatus.checks.internet.connected ? 'ESTABLISHED' : 'DISCONNECTED'}
        </div>
        <div class="metric-sub">
          {#if healthStore.healthStatus.checks.internet.connected}
            Latency: {healthStore.healthStatus.checks.internet.latency}
          {:else}
            No route to the internet. Syncs will fail until the connection is restored.
          {/if}
        </div>
      </div>

      <!-- Storage -->
      <div class="metric-card">
        <h3 class="metric-title">STORAGE LOAD</h3>
        <div class="metric-value">
          {healthStore.healthStatus.checks.disk.used_percent}
        </div>
        <div class="metric-sub">
          {formatBytes(healthStore.healthStatus.checks.disk.free_bytes)} free of {formatBytes(healthStore.healthStatus.checks.disk.total_bytes)}
        </div>
        {#if parseFloat(healthStore.healthStatus.checks.disk.used_percent) > 90}
          <div class="metric-sub" style="color: var(--error); margin-top: 8px;">
            Almost full. Free up space or syncs will start failing.
          </div>
        {/if}
        <!-- Mini progress bar -->
        <div class="progress-bar-bg" style="margin-top: 16px;">
          <div class="progress-bar-fill" style="width: {healthStore.healthStatus.checks.disk.used_percent};"></div>
        </div>
      </div>

      <!-- Database -->
      <div class="metric-card">
        <h3 class="metric-title">DATABASE CLUSTER</h3>
        <div class="metric-value" style="color: {healthStore.healthStatus.checks.database ? 'var(--primary)' : 'var(--error)'};">
          {healthStore.healthStatus.checks.database ? 'ONLINE' : 'UNREACHABLE'}
        </div>
        <div class="metric-sub">
          {#if healthStore.healthStatus.checks.database}
            SQLite Core Engine
          {:else}
            The database did not respond. Check the server logs and free disk space.
          {/if}
        </div>
      </div>

      <!-- Workers -->
      <div class="metric-card">
        <h3 class="metric-title">SYNC WORKERS</h3>
        <div class="metric-value">
          {healthStore.healthStatus.checks.workers.active_tasks} / {healthStore.healthStatus.checks.workers.total_workers}
        </div>
        <div class="metric-sub">
          {healthStore.healthStatus.checks.workers.queued_tasks} tasks queued
        </div>
      </div>
    {:else}
      <div class="metric-card" style="grid-column: span 2; text-align: center;">
        <p style="color: var(--on-surface-variant);">
          {healthStore.unreachable ? "Can't reach the GitPatrol server. Check that it's running, then refresh the page." : 'Awaiting telemetry data...'}
        </p>
      </div>
    {/if}
  </div>

  <div class="incidents-section">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px;">
      <h2 style="font-size: 1.5rem; margin: 0;">Security & Sync Logs</h2>
      {#if incidentsStore.incidents.length > 0}
        <div class="badge status-critical" style="font-size: 0.7rem; padding: 4px 12px;">
          <span class="dot"></span>
          {incidentsStore.incidents.length} ACTIVE ALERTS
        </div>
      {/if}
    </div>

    {#if incidentsStore.incidents.length === 0}
      <div style="text-align: center; padding: 60px 20px; background: var(--surface-container-low); border-radius: 24px; border: 1px solid var(--glass-border);">
        <Icon name="activity" size={48} stroke="var(--on-surface-variant)" style="margin-bottom: 16px; opacity: 0.5;" />
        <h3 style="font-size: 1.2rem; margin-bottom: 8px;">All Systems Nominal</h3>
        <p style="color: var(--on-surface-variant);">No active incidents or security alerts detected across the fleet.</p>
      </div>
    {:else}
      <div class="incidents-list">
        {#each incidentsStore.incidents as incident (incident.id)}
          <div class="incident-row">
            <div style="display: flex; align-items: center; gap: 16px;">
              <div class="alert-icon">
                <Icon name="alert-circle" size={20} stroke="var(--error)" />
              </div>
              <div>
                <div style="font-weight: 700; font-size: 1rem; margin-bottom: 4px;">
                  {incident.repo_name || 'SYSTEM ALERT'}
                </div>
                <div style="color: var(--on-surface-variant); font-size: 0.85rem;">
                  {incident.message}
                </div>
              </div>
            </div>
            
            <div style="display: flex; align-items: center; gap: 24px;">
              <div style="font-family: 'Space Grotesk', sans-serif; font-size: 0.8rem; color: var(--on-surface-variant);">
                {new Date(incident.created_at).toLocaleString()}
              </div>
              <button class="secondary" style="padding: 8px 16px; font-size: 0.7rem;" onclick={() => clearIncident(incident.id)}>
                ACKNOWLEDGE
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .header {
    padding: 40px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  .status-chip {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    border-radius: 999px;
    font-size: 0.8rem;
    font-weight: 700;
    letter-spacing: 0.1em;
  }
  
  .status-healthy { color: var(--primary); background: rgba(77, 221, 187, 0.1); }
  .status-degraded { color: var(--warning, #ffcc00); background: rgba(255, 204, 0, 0.1); }
  .status-critical { color: var(--error); background: var(--error-container); }
  .status-offline { color: var(--on-surface-variant); background: var(--surface-container-highest); }
  
  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: currentColor;
    box-shadow: 0 0 8px currentColor;
  }

  .metrics-grid {
    padding: 0 40px 80px 40px;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 24px;
    max-width: 1200px;
  }

  .metric-card {
    background: var(--surface-container-low);
    border-radius: 24px;
    padding: 32px;
    display: flex;
    flex-direction: column;
  }

  .metric-title {
    font-size: 0.7rem;
    font-weight: 800;
    letter-spacing: 0.1em;
    color: var(--on-surface-variant);
    margin: 0 0 16px 0;
  }

  .metric-value {
    font-size: 2.5rem;
    font-weight: 700;
    font-family: 'Space Grotesk', sans-serif;
    color: var(--on-surface);
    line-height: 1;
    margin-bottom: 8px;
  }

  .metric-sub {
    font-size: 0.9rem;
    color: var(--on-surface-variant);
  }

  .progress-bar-bg {
    height: 6px;
    background: var(--surface-container-highest);
    border-radius: 3px;
    overflow: hidden;
  }

  .progress-bar-fill {
    height: 100%;
    background: var(--primary);
    border-radius: 3px;
    box-shadow: 0 0 12px rgba(77, 221, 187, 0.4);
  }

  .incidents-section {
    padding: 0 40px 80px 40px;
    max-width: 1200px;
  }

  .incidents-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .incident-row {
    background: var(--surface-container-low);
    border: 1px solid var(--glass-border);
    border-radius: 16px;
    padding: 20px 24px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    transition: background 0.2s;
  }

  .incident-row:hover {
    background: var(--surface-container-highest);
  }

  .alert-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    background: var(--error-container);
    border-radius: 50%;
  }
</style>
