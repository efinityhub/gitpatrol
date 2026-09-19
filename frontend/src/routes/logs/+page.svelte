<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { fade } from 'svelte/transition';
  import Icon from '$lib/components/Icon.svelte';
  import { apiFetch } from '$lib/client';
  import { browser } from '$app/environment';

  type Syslog = {
    id: number;
    level: string;
    message: string;
    attributes: Record<string, any>;
    created_at: string;
    time?: string; // from websocket
  };

  let logs: Syslog[] = $state([]);
  let loading = $state(true);
  let ws: WebSocket | null = null;
  let logsContainer: HTMLElement | null = $state(null);
  let autoScroll = $state(true);
  let isPaused = $state(false);
  let pausedBuffer: Syslog[] = [];

  // Filter states
  let filterLevel = $state('ALL');
  let searchQuery = $state('');

  onMount(async () => {
    // 1. Fetch historical logs
    try {
      const res = await apiFetch('/api/logs?limit=500');
      const data = await res.json();
      // Backend returns them DESC, we want ASC for terminal flow
      logs = data.reverse();
    } catch (e) {
      console.error('Failed to load logs:', e);
    } finally {
      loading = false;
      scrollToBottom();
    }

    // 2. Connect WebSocket for live logs
    if (browser) {
      connectWebSocket();
    }
  });

  onDestroy(() => {
    if (ws) {
      ws.close();
    }
  });

  function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    // During dev, Vite proxies to 8080. But in prod, it's relative.
    const wsUrl = import.meta.env.DEV 
      ? `ws://localhost:8080/ws`
      : `${protocol}//${window.location.host}/ws`;

    ws = new WebSocket(wsUrl);

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === 'syslog') {
          const newLog = data.log;
          const logEntry = {
            id: Date.now(), // dummy id for list
            level: newLog.level,
            message: newLog.message,
            attributes: newLog.attributes,
            created_at: newLog.time
          };

          if (isPaused) {
            pausedBuffer.push(logEntry);
            if (pausedBuffer.length > 1000) {
              pausedBuffer = pausedBuffer.slice(pausedBuffer.length - 1000);
            }
          } else {
            logs = [...logs, logEntry];
            if (logs.length > 1000) {
              logs = logs.slice(logs.length - 1000);
            }
            if (autoScroll) {
              setTimeout(scrollToBottom, 10);
            }
          }
        }
      } catch (e) {
        console.error("WS Parse error", e);
      }
    };

    ws.onclose = () => {
      setTimeout(connectWebSocket, 5000);
    };
  }

  function togglePause() {
    isPaused = !isPaused;
    if (!isPaused && pausedBuffer.length > 0) {
      logs = [...logs, ...pausedBuffer];
      if (logs.length > 1000) {
        logs = logs.slice(logs.length - 1000);
      }
      pausedBuffer = [];
      if (autoScroll) {
        setTimeout(scrollToBottom, 10);
      }
    }
  }

  function scrollToBottom() {
    if (logsContainer) {
      logsContainer.scrollTop = logsContainer.scrollHeight;
    }
  }

  function handleScroll() {
    if (!logsContainer) return;
    const isAtBottom = logsContainer.scrollHeight - logsContainer.scrollTop <= logsContainer.clientHeight + 50;
    autoScroll = isAtBottom;
  }

  function formatTime(timestamp: string) {
    if (!timestamp) return '';
    const d = new Date(timestamp);
    return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}:${d.getSeconds().toString().padStart(2, '0')}.${d.getMilliseconds().toString().padStart(3, '0')}`;
  }

  function getLevelColor(level: string) {
    switch (level?.toUpperCase()) {
      case 'ERROR': return 'var(--error)';
      case 'WARN': return 'var(--warning)';
      case 'INFO': return '#7aa2f7'; // A nice blue for INFO
      case 'DEBUG': return 'var(--text-muted)';
      default: return 'var(--text)';
    }
  }

  let filteredLogs = $derived(logs.filter(log => {
    if (filterLevel !== 'ALL' && log.level !== filterLevel) return false;
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      const matchMsg = log.message?.toLowerCase().includes(query);
      const matchAttrs = JSON.stringify(log.attributes).toLowerCase().includes(query);
      return matchMsg || matchAttrs;
    }
    return true;
  }));

</script>

<div class="page-container">
  <div class="header-section">
    <div class="title-group">
      <h1>System Logs</h1>
      <p class="subtitle">Live backend telemetry and activity</p>
    </div>

    <div class="controls">
      <div class="search-box">
        <Icon name="search" size={16} />
        <input 
          type="text" 
          placeholder="Grep logs..." 
          bind:value={searchQuery}
        />
      </div>

      <div class="level-filters">
        {#each ['ALL', 'INFO', 'WARN', 'ERROR'] as level}
          <button 
            class="filter-btn" 
            class:active={filterLevel === level}
            onclick={() => filterLevel = level}
          >
            {level}
          </button>
        {/each}
      </div>
    </div>
  </div>

  <div class="terminal-wrapper">
    <div class="terminal-header">
      <div class="mac-dots">
        <span></span><span></span><span></span>
      </div>
      <div class="terminal-title">gitpatrol-backend — slog</div>
      <div class="terminal-actions">
        <button 
          class="play-pause-btn" 
          class:active={!isPaused}
          class:paused={isPaused}
          onclick={togglePause}
          data-tooltip-bottom={isPaused ? "Resume Live Feed" : "Pause Live Feed"}
        >
          <Icon name={isPaused ? 'play' : 'pause'} size={14} />
          {#if isPaused && pausedBuffer.length > 0}
            <span class="buffer-badge">{pausedBuffer.length}</span>
          {/if}
        </button>
        <div class="divider"></div>
        <button 
          class="scroll-lock" 
          class:active={autoScroll} 
          onclick={() => { autoScroll = !autoScroll; if (autoScroll) scrollToBottom(); }}
          data-tooltip-bottom="Auto-scroll"
        >
          <Icon name={autoScroll ? 'lock' : 'unlock'} size={14} />
        </button>
        <button class="clear-btn" onclick={() => logs = []} data-tooltip-bottom="Clear Terminal">
          <Icon name="trash" size={14} />
        </button>
      </div>
    </div>

    <div 
      class="terminal-body" 
      bind:this={logsContainer}
      onscroll={handleScroll}
    >
      {#if loading}
        <div class="loading-state">
          <div class="spinner"></div>
          <span>Loading historical logs...</span>
        </div>
      {:else if filteredLogs.length === 0}
        <div class="empty-state">
          No logs matching current filters.
        </div>
      {:else}
        {#each filteredLogs as log (log.id)}
          <div class="log-line" in:fade={{ duration: 150 }}>
            <span class="log-time">{formatTime(log.created_at)}</span>
            <span class="log-level" style="color: {getLevelColor(log.level)}">
              [{log.level || 'UNKNOWN'}]
            </span>
            <span class="log-msg">{log.message}</span>
            
            {#if log.attributes && Object.keys(log.attributes).length > 0}
              <span class="log-attrs">
                {#each Object.entries(log.attributes) as [k, v]}
                  <span class="attr"><span class="attr-key">{k}=</span><span class="attr-val">{typeof v === 'object' ? JSON.stringify(v) : String(v)}</span></span>
                {/each}
              </span>
            {/if}
          </div>
        {/each}
      {/if}
    </div>
  </div>
</div>

<style>
  .page-container {
    padding: 0 40px 40px 40px;
    max-width: 1600px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    height: 100%;
    box-sizing: border-box;

    @media (max-width: 1023px) { padding: 0 24px 24px 24px; }
    @media (max-width: 767px) { padding: 0 16px 24px 16px; }
  }

  .header-section {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
    flex-wrap: wrap;
    gap: 16px;

    @media (max-width: 767px) {
      flex-direction: column;
      align-items: flex-start;
    }
  }

  .title-group h1 {
    font-size: 32px;
    font-weight: 700;
    margin: 0 0 4px 0;
    background: linear-gradient(180deg, #FFFFFF 0%, rgba(255, 255, 255, 0.7) 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .subtitle {
    margin: 0;
    color: var(--text-muted);
    font-size: 14px;
  }

  .controls {
    display: flex;
    gap: 16px;
    align-items: center;
    flex-wrap: wrap;
  }

  .search-box {
    position: relative;
    display: flex;
    align-items: center;
    color: var(--text-muted);

    input {
      background: var(--surface-container);
      border: 1px solid var(--surface-border);
      border-radius: 8px;
      padding: 8px 12px 8px 36px;
      color: var(--text);
      font-size: 14px;
      width: 200px;
      transition: all 0.2s ease;

      &:focus {
        outline: none;
        border-color: var(--primary);
        width: 250px;
      }
    }

    :global(svg) {
      position: absolute;
      left: 12px;
    }
  }

  .level-filters {
    display: flex;
    background: var(--surface-container);
    padding: 4px;
    border-radius: 8px;
    border: 1px solid var(--surface-border);
  }

  .filter-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    padding: 6px 12px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;

    &:hover {
      color: var(--text);
    }

    &.active {
      background: var(--surface-container-high);
      color: var(--text);
    }
  }

  .terminal-wrapper {
    flex: 1;
    display: flex;
    flex-direction: column;
    background: #0D0D0F;
    border: 1px solid var(--surface-border);
    border-radius: 12px;
    overflow: hidden;
    min-height: 500px;
    box-shadow: 0 12px 32px rgba(0,0,0,0.4);
  }

  .terminal-header {
    background: #1A1A1E;
    padding: 12px 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid rgba(255,255,255,0.05);
  }

  .mac-dots {
    display: flex;
    gap: 6px;

    span {
      width: 12px;
      height: 12px;
      border-radius: 50%;
      background: #333;
    }
    span:nth-child(1) { background: #FF5F56; }
    span:nth-child(2) { background: #FFBD2E; }
    span:nth-child(3) { background: #27C93F; }
  }

  .terminal-title {
    color: #888;
    font-size: 13px;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-weight: 500;
  }

  .terminal-actions {
    display: flex;
    gap: 8px;
    
    button {
      background: transparent;
      border: none;
      color: #666;
      cursor: pointer;
      padding: 4px;
      border-radius: 4px;
      display: flex;
      align-items: center;
      justify-content: center;
      transition: all 0.2s;
      position: relative;

      &:hover {
        color: #CCC;
        background: rgba(255,255,255,0.1);
      }

      &.active {
        color: var(--primary);
      }

      &.paused {
        color: #FFBD2E;
      }
    }
    
    .divider {
      width: 1px;
      height: 16px;
      background: rgba(255,255,255,0.1);
      margin: 0 4px;
    }
    
    .buffer-badge {
      position: absolute;
      top: -6px;
      right: -6px;
      background: var(--primary);
      color: var(--surface-container-lowest);
      font-size: 9px;
      font-weight: 700;
      padding: 2px 4px;
      border-radius: 8px;
    }
  }

  .terminal-body {
    flex: 1;
    padding: 16px;
    overflow-y: auto;
    font-family: 'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, Monaco, monospace;
    font-size: 13px;
    line-height: 1.6;
    scroll-behavior: smooth;

    &::-webkit-scrollbar {
      width: 10px;
    }
    &::-webkit-scrollbar-track {
      background: #0D0D0F;
    }
    &::-webkit-scrollbar-thumb {
      background: #333;
      border-radius: 5px;
      border: 2px solid #0D0D0F;
    }
  }

  .log-line {
    margin-bottom: 4px;
    word-break: break-word;
    display: flex;
    flex-wrap: wrap;
    column-gap: 8px;
    
    &:hover {
      background: rgba(255,255,255,0.02);
    }
  }

  .log-time {
    color: #666;
    min-width: 95px;
  }

  .log-level {
    font-weight: 700;
    min-width: 60px;
  }

  .log-msg {
    color: #E2E2E2;
  }

  .log-attrs {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    margin-left: 8px;
  }

  .attr {
    font-size: 12px;
  }

  .attr-key {
    color: #888;
  }

  .attr-val {
    color: #A3D8A5; /* slightly green for values */
  }

  .loading-state, .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #666;
    gap: 16px;
  }

  .spinner {
    width: 24px;
    height: 24px;
    border: 2px solid #333;
    border-top-color: var(--primary);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
