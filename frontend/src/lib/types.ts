export interface Repository {
  id: number;
  name: string;
  url: string;
  interval_minutes: number;
  last_sync: string;
  status: string;
  last_commit: string;
  error_message: string;
  stars: number;
  forks: number;
  open_issues: number;
  commit_history: string;
  health_score: number;
  default_branch: string;
  auto_patrol: number;
  size_bytes: number;
  progress?: number;
}

export interface Incident {
  id: number;
  repo_id: number | null;
  repo_name: string;
  message: string;
  created_at: string;
  resolved: number;
}

export interface User {
  id: number;
  username: string;
}

export interface Toast {
  id: number;
  message: string;
  type: 'success' | 'error' | 'info';
}

export interface Commit {
  hash: string;
  author: string;
  date: string;
  message: string;
  branch: string;
}

// GitHub API types (raw responses stored by backend)

export interface GitHubUser {
  login: string;
  id: number;
  avatar_url?: string;
  html_url?: string;
}

export interface Issue {
  id: number;
  node_id: string;
  number: number;
  title: string;
  user: GitHubUser | null;
  labels: unknown[];
  state: 'open' | 'closed';
  locked: boolean;
  assignees: GitHubUser[];
  milestone: unknown | null;
  comments: number;
  created_at: string;
  updated_at: string;
  closed_at: string | null;
  body: string | null;
}

export interface Release {
  id: number;
  tag_name: string;
  target_commitish: string;
  name: string | null;
  draft: boolean;
  prerelease: boolean;
  created_at: string;
  published_at: string;
  author: GitHubUser;
  body: string | null;
  assets: unknown[];
  tarball_url: string;
  zipball_url: string;
}

// Health endpoint types

export interface HealthInternetCheck {
  connected: boolean;
  latency: string;
}

export interface HealthDiskCheck {
  free_bytes: number;
  total_bytes: number;
  used_percent: string;
}

export interface HealthWorkerCheck {
  active_tasks: number;
  queued_tasks: number;
  total_workers: number;
}

export interface HealthUptimeCheck {
  percent: number | null;
  window_days: number;
}

export interface HealthChecks {
  internet: HealthInternetCheck;
  disk: HealthDiskCheck;
  database: boolean;
  workers: HealthWorkerCheck;
  uptime?: HealthUptimeCheck;
}

export interface HealthStatus {
  status: 'healthy' | 'degraded' | 'critical' | 'offline';
  checks: HealthChecks;
  timestamp?: string;
}
