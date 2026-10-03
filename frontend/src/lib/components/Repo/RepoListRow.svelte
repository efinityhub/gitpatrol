<script lang="ts">
  import { type Repository } from '$lib/types';
  import { getAvatarUrl, getRemainingTime, isSyncing, handleAvatarError, humanToMinutes, minutesToHuman } from '$lib/utils';
  import { goto } from '$app/navigation';

  let { repo } = $props<{ repo: Repository }>();
</script>

<div class="list-item" onclick={() => goto(`/repositories/${repo.id}`)} onkeydown={(e) => e.key === 'Enter' && goto(`/repositories/${repo.id}`)} role='button' tabindex=0>
  <div style="display: flex; align-items: center; gap: 24px;">
    <div class="radial-timer" class:is-syncing={isSyncing(repo)} data-tooltip={getRemainingTime(repo)} style="width: 32px; height: 32px;">
      <svg width="32" height="32">
        <circle cx="16" cy="16" r="12" />
        <circle 
          cx="16" cy="16" r="12" 
          class="progress" 
          class:active-pulse={repo.status !== 'syncing' && repo.auto_patrol === 1} 
          style="stroke-dasharray: 75; stroke-dashoffset: {isSyncing(repo) ? 0 : 75 - (repo.progress || 0) * 0.75}" 
        />
      </svg>
    </div>
    <img 
      src={getAvatarUrl(repo.url)} 
      alt="" 
      style="width: 32px; height: 32px; border-radius: 8px; border: none; background: var(--surface-container-highest);"
      onerror={handleAvatarError}
    />
    <div>
      <div style="display: flex; align-items: center; gap: 12px;">
        <div style="font-weight: 700; font-size: 1.1rem;">{repo.name}</div>
        {#if repo.status === 'synced'}
          <span class="badge status-healthy" style="font-size: 0.6rem; padding: 2px 8px;">SYNCED</span>
        {:else if repo.status === 'error'}
          <span class="badge status-critical" style="font-size: 0.6rem; padding: 2px 8px;">ERROR</span>
        {/if}
      </div>
      <div style="font-size: 0.8rem; color: var(--on-surface-variant);">{repo.url.replace('https://github.com/', '')}</div>
      {#if repo.status === 'error' && repo.error_message}
        <div style="font-size: 0.8rem; color: var(--error); margin-top: 4px;">{repo.error_message}</div>
      {/if}
    </div>
  </div>

  <div style="display: flex; gap: 40px; align-items: center;">
    <div class="stat-item">
      <span>Health</span>
      <b>{repo.health_score}%</b>
    </div>
    <div class="stat-item">
      <span>Interval</span>
      <b>{minutesToHuman(repo.interval_minutes)}</b>
    </div>
  </div>
</div>
