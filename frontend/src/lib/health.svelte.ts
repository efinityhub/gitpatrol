import { apiFetch } from './client';
import { authStore } from './auth.svelte';
import type { HealthStatus } from './types';

class HealthStore {
  healthStatus = $state<HealthStatus | null>(null);
  unreachable = $state(false);
  private healthTimer: ReturnType<typeof setInterval> | null = null;

  constructor() {
    if (typeof window !== 'undefined') {
      window.addEventListener('gitpatrol:logged_out', () => {
        this.healthStatus = null;
        this.unreachable = false;
        if (this.healthTimer) {
          clearInterval(this.healthTimer);
          this.healthTimer = null;
        }
      });
    }
  }

  async fetchHealth() {
    try {
      const res = await apiFetch('/api/health');
      if (res.ok) {
        this.healthStatus = await res.json();
        this.unreachable = false;
      } else {
        this.healthStatus = null;
        this.unreachable = true;
      }
    } catch (e) {
      this.healthStatus = null;
      this.unreachable = true;
    }
  }

  setupHealthPolling() {
    if (this.healthTimer) clearInterval(this.healthTimer);
    this.healthTimer = setInterval(() => {
      if (authStore.isAuthenticated) {
        this.fetchHealth();
      }
    }, 60000); // Fallback polling every 1 minute
  }
}

export const healthStore = new HealthStore();
