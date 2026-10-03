<script lang="ts">
  import { type Repository } from '$lib/types';
  import { getAvatarUrl, getRemainingTime, getProgress, isSyncing, minutesToHuman, handleAvatarError } from '$lib/utils';
  import Icon from '$lib/components/Icon.svelte';
  import { goto } from '$app/navigation';

  let { repo } = $props<{ repo: Repository }>();
</script>

<div class="card" onclick={() => goto(`/repositories/${repo.id}`)} onkeydown={(e) => e.key === 'Enter' && goto(`/repositories/${repo.id}`)} role='button' tabindex=0>
  <div style="position: absolute; top: 32px; right: 32px; display: flex; align-items: center; gap: 16px;">
    <div 
      class="radial-timer" 
      class:is-syncing={isSyncing(repo)}
      data-tooltip={getRemainingTime(repo)}
    >
      <svg width="40" height="40">
        <circle cx="20" cy="20" r="16" />
        <circle 
          class="progress" 
          class:active-pulse={repo.status === 'syncing'}
          cx="20" cy="20" r="16" 
          stroke-dasharray="100.5" 
          stroke-dashoffset={100.5 - (repo.progress || 0)} 
        />
      </svg>
    </div>
    <div class="health-score" data-tooltip="Tactical Health Score" style="color: {repo.health_score > 70 ? 'var(--primary)' : 'var(--warning, #ffcc00)'}; border-color: {repo.health_score > 70 ? 'var(--primary)' : repo.health_score > 40 ? 'var(--warning, #ffcc00)' : 'var(--error)'}44">
      {repo.health_score}
    </div>
  </div>

  <div style="display: flex; align-items: center; gap: 24px;">
    <img 
      src={getAvatarUrl(repo.url)} 
      alt="" 
      style="width: 44px; height: 44px; border-radius: 12px; background: var(--surface-container-highest);"
      onerror={handleAvatarError}
    />
    <div>
      <h3 style="font-size: 1.5rem; margin: 0; letter-spacing: -0.02em;">{repo.name}</h3>
      <div style="display: flex; align-items: center; gap: 8px; margin-top: 8px;">
        {#if repo.status === 'syncing'}
          <div class="badge status-healthy" style="font-size: 0.6rem; padding: 2px 8px;">
            <span class="dot"></span>
            SYNCING
          </div>
        {:else if repo.status === 'error'}
          <div class="badge status-critical" style="font-size: 0.6rem; padding: 2px 8px;">
            <span class="dot"></span>
            FAILED
          </div>
        {:else}
          <div class="badge status-healthy" style="font-size: 0.6rem; padding: 2px 8px; color: var(--primary); background: rgba(77, 221, 187, 0.1);">
            <span class="dot"></span>
            SECURED
          </div>
        {/if}
        <span class="branch-badge">{repo.default_branch}</span>
      </div>
    </div>
  </div>

  {#if repo.status === 'error' && repo.error_message}
    <div style="display: flex; align-items: flex-start; gap: 8px; margin-top: 16px; padding: 10px 12px; border-radius: 8px; background: var(--error-container); color: var(--error); font-size: 0.8rem; line-height: 1.4;">
      <Icon name="alert-circle" size={14} strokeWidth={3} style="flex-shrink: 0; margin-top: 2px;" />
      <span>{repo.error_message}</span>
    </div>
  {/if}

  <div class="stats-row">
    <div class="stat-item" data-tooltip="GitHub Stars">
      <div style="display: flex; gap: .25rem; align-items: center;">
        <Icon name="star" size={14} fill="#ffcc00" stroke="none" style="opacity: 0.8;" /><b>{repo.stars}</b>
      </div>
    </div>
    <div class="stat-item" data-tooltip="Open Issues">
      <div style="display: flex; gap: .25rem; align-items: center;">
        <Icon name="alert-circle" size={14} strokeWidth={3} /><b>{repo.open_issues}</b>
      </div>
    </div>
  </div>

  <div class="mini-chart">
    {#each JSON.parse(repo.commit_history || '[]') as count}
      <div class="chart-bar" style="height: {count === 0 ? '4px' : Math.min(100, (count / 10) * 100)}%; background: {count === 0 ? 'var(--surface-container-highest)' : `rgba(77, 221, 187, ${0.3 + (Math.min(count, 10) / 10) * 0.7})`}; box-shadow: {count > 5 ? `0 0 12px rgba(77, 221, 187, ${(Math.min(count, 10) / 10) * 0.4})` : 'none'};" data-tooltip="{count} commits"></div>
    {/each}
  </div>

  <div class="info-row">
    <div class="info-item">
      <span class="label">Last Patrol</span>
      <span>{repo.last_sync ? new Date(repo.last_sync).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : 'NEVER'}</span>
    </div>
    <div class="info-item" style="text-align: right;">
      <span class="label">Interval</span>
      <span>{minutesToHuman(repo.interval_minutes)}</span>
    </div>
  </div>
</div>
