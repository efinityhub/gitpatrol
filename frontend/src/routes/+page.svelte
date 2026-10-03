<script lang="ts">
  import { fade } from 'svelte/transition';
  import { reposStore } from '$lib/repos.svelte';
  import { healthStore } from '$lib/health.svelte';
  import { formatBytes, formatUptime, getProgress } from '$lib/utils';
  import { type Repository } from '$lib/types';
  import { onMount } from 'svelte';

  // Components
  import Header from '$lib/components/Layout/Header.svelte';
  import RepoCard from '$lib/components/Repo/RepoCard.svelte';
  import RepoListRow from '$lib/components/Repo/RepoListRow.svelte';
  import AddRepoModal from '$lib/components/Modals/AddRepoModal.svelte';
  import IncidentModal from '$lib/components/Modals/IncidentModal.svelte';
  import Icon from '$lib/components/Icon.svelte';

  let viewMode = $state<'grid' | 'list'>('grid');
  let showAddModal = $state(false);
  let showIncidentModal = $state(false);
  
  // Search & Filter state
  let searchQuery = $state('');
  let filterStatus = $state('all');

  onMount(() => {
    const timer = setInterval(() => {
      reposStore.repositories = reposStore.repositories.map(r => ({ ...r, progress: getProgress(r) }));
    }, 1000);
    return () => clearInterval(timer);
  });

  let protectedBytes = $derived(reposStore.repositories.reduce((sum, repo) => sum + (repo.size_bytes || 0), 0));
  let uptime = $derived(healthStore.healthStatus?.checks.uptime);

  // Derived state for filtered repositories
  let filteredRepos = $derived(
    reposStore.repositories.filter(repo => {
      const matchesSearch = repo.name.toLowerCase().includes(searchQuery.toLowerCase());
      
      let matchesFilter = true;
      if (filterStatus === 'syncing') {
        matchesFilter = repo.status === 'syncing';
      } else if (filterStatus === 'secured') {
        matchesFilter = repo.status === 'synced';
      } else if (filterStatus === 'failed') {
        matchesFilter = repo.status === 'error';
      }
      
      return matchesSearch && matchesFilter;
    })
  );
</script>

<div class="page-container" transition:fade>
  <Header bind:viewMode bind:showIncidentModal bind:searchQuery />
  
  <div class="hero">
    <div class="hero-stats">
      <div class="stat-block">
        <span class="stat-label">UPTIME ({uptime?.window_days ?? 30}D)</span>
        <span class="stat-value">{formatUptime(uptime?.percent)}</span>
      </div>
      <div class="stat-block">
        <span class="stat-label">TOTAL PROTECTED</span>
        <span class="stat-value">{formatBytes(protectedBytes, 1)}</span>
      </div>
      <div class="stat-block">
        <span class="stat-label">ACTIVE MIRRORS</span>
        <span class="stat-value">{filteredRepos.length}/{reposStore.repositories.length}</span>
      </div>
    </div>
    
    <div class="hero-filters">
      <div class="tab-container" style="width: auto; margin-bottom: 0;">
        <button class="tab-btn" class:active={filterStatus === 'all'} onclick={() => filterStatus = 'all'}>ALL</button>
        <button class="tab-btn" class:active={filterStatus === 'syncing'} onclick={() => filterStatus = 'syncing'}>SYNCING</button>
        <button class="tab-btn" class:active={filterStatus === 'secured'} onclick={() => filterStatus = 'secured'}>SECURED</button>
        <button class="tab-btn" class:active={filterStatus === 'failed'} onclick={() => filterStatus = 'failed'}>FAILED</button>
      </div>
    </div>
  </div>

  <div class="content-area">
    {#if viewMode === 'grid'}
      <div class="repo-grid">
        {#each filteredRepos as repo (repo.id)}
          <RepoCard {repo} />
        {/each}
      </div>
    {:else}
      <div class="repo-list">
        {#each filteredRepos as repo (repo.id)}
          <RepoListRow {repo} />
        {/each}
      </div>
    {/if}

    {#if filteredRepos.length === 0}
      <div style="text-align: center; padding: 120px 40px; background: var(--surface-container-low); border-radius: 32px;">
        <h2 style="font-size: 2rem; margin-bottom: 16px;">No Repositories Found</h2>
        <p style="color: var(--on-surface-variant); margin-bottom: 32px;">Try adjusting your search or filters.</p>
        <button onclick={() => showAddModal = true}>ADD REPOSITORY</button>
      </div>
    {/if}
  </div>

  <div class="fab" data-tooltip="Deploy New Patrol">
    <button onclick={() => showAddModal = true} aria-label="Deploy New Patrol">
      <Icon name="plus" size={28} stroke="white" strokeWidth={3} />
    </button>
  </div>
</div>

{#if showAddModal}
  <AddRepoModal bind:show={showAddModal} />
{/if}

{#if showIncidentModal}
  <IncidentModal bind:show={showIncidentModal} />
{/if}

<style>
  .page-container {
    display: flex;
    flex-direction: column;
  }
  .hero {
    padding: 0 40px 40px 40px;
    display: flex;
    justify-content: space-between;
    align-items: flex-end;

    @media (max-width: 1023px) { padding: 0 24px 24px 24px; }
    @media (max-width: 767px) {
      padding: 0 16px 24px 16px;
      flex-direction: column;
      align-items: flex-start;
      gap: 24px;
    }
  }
  .hero-stats {
    display: flex;
    gap: 40px;

    @media (max-width: 767px) { gap: 24px; width: 100%; justify-content: space-between; }
  }
  .stat-block {
    display: flex;
    flex-direction: column;
  }
  .stat-label {
    font-size: 0.65rem;
    font-weight: 800;
    letter-spacing: 0.1em;
    color: var(--on-surface-variant);
    margin-bottom: 4px;
  }
  .stat-value {
    font-size: 2.5rem;
    font-weight: 700;
    font-family: 'Space Grotesk', sans-serif;
    color: var(--on-surface);
    line-height: 1;
  }
  .content-area {
    padding: 0 40px 80px 40px;

    @media (max-width: 1023px) { padding: 0 24px 60px 24px; }
    @media (max-width: 767px) { padding: 0 16px 40px 16px; }
  }
</style>
