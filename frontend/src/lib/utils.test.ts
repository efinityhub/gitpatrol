import { describe, expect, it } from 'vitest';
import {
	describeHealth,
	formatBytes,
	formatUptime,
	getNormalizedUrl,
	getRemainingTime,
	humanToMinutes,
	minutesToHuman,
	suggestName
} from './utils';
import type { HealthStatus, Repository } from './types';

describe('getNormalizedUrl', () => {
	it.each([
		['https://github.com/efinityhub/gitpatrol', 'https://github.com/efinityhub/gitpatrol'],
		['https://github.com/efinityhub/gitpatrol.git', 'https://github.com/efinityhub/gitpatrol'],
		['  https://github.com/efinityhub/gitpatrol  ', 'https://github.com/efinityhub/gitpatrol'],
		['git@github.com:efinityhub/gitpatrol.git', 'https://github.com/efinityhub/gitpatrol'],
		['git@gitlab.com:group/project.git', 'https://gitlab.com/group/project'],
		['https://github.com/efinityhub/gitpatrol/tree/main/backend', 'https://github.com/efinityhub/gitpatrol'],
		['https://github.com/efinityhub/gitpatrol/blob/main/README.md', 'https://github.com/efinityhub/gitpatrol'],
		['efinityhub/gitpatrol', 'https://github.com/efinityhub/gitpatrol'],
		['https://gitlab.com/group/sub/project', 'https://gitlab.com/group/sub/project']
	])('%s -> %s', (input, expected) => {
		expect(getNormalizedUrl(input)).toBe(expected);
	});

	it.each(['', 'ab', undefined as unknown as string])('returns empty for %j', (input) => {
		expect(getNormalizedUrl(input)).toBe('');
	});

	it('leaves a bare word alone instead of guessing a provider', () => {
		expect(getNormalizedUrl('gitpatrol')).toBe('gitpatrol');
	});

	it('does not expand owner/repo/extra shorthand', () => {
		expect(getNormalizedUrl('a/b/c')).toBe('a/b/c');
	});
});

describe('suggestName', () => {
	it.each([
		['https://github.com/efinityhub/gitpatrol', 'Gitpatrol'],
		['git@github.com:efinityhub/gitpatrol.git', 'Gitpatrol'],
		['https://github.com/efinityhub/gitpatrol/tree/main', 'Gitpatrol'],
		['efinityhub/gitpatrol', 'Gitpatrol'],
		['https://gitlab.com/group/sub/my-project.git', 'My-project']
	])('%s -> %s', (input, expected) => {
		expect(suggestName(input)).toBe(expected);
	});

	it('returns empty for empty input', () => {
		expect(suggestName('')).toBe('');
	});
});

describe('minutesToHuman / humanToMinutes', () => {
	it.each([
		[0, '0m'],
		[-5, '0m'],
		[45, '45m'],
		[60, '1h'],
		[90, '1h 30m'],
		[1440, '1d'],
		[1500, '1d 1h'],
		[2881, '2d 1m']
	])('%i minutes -> %s', (minutes, expected) => {
		expect(minutesToHuman(minutes)).toBe(expected);
	});

	it.each([
		['45m', 45],
		['1h', 60],
		['1h 30m', 90],
		['1d', 1440],
		['2d 1h 5m', 2945],
		['1D 2H', 1560]
	])('parses %s as %i minutes', (text, expected) => {
		expect(humanToMinutes(text)).toBe(expected);
	});

	it.each(['', 'soon', '0m'])('falls back to 60 minutes for %j', (text) => {
		expect(humanToMinutes(text)).toBe(60);
	});

	it('round-trips', () => {
		for (const minutes of [1, 59, 60, 61, 1439, 1440, 10080]) {
			expect(humanToMinutes(minutesToHuman(minutes))).toBe(minutes);
		}
	});
});

describe('formatBytes', () => {
	it.each([
		[0, '0 Bytes'],
		[512, '512 Bytes'],
		[1024, '1 KB'],
		[1536, '1.5 KB'],
		[1024 * 1024, '1 MB'],
		[7_864_320, '7.5 MB'],
		[1024 ** 3, '1 GB'],
		[1024 ** 5, '1 PB'],
		[1024 ** 6, '1024 PB']
	])('%i -> %s', (bytes, expected) => {
		expect(formatBytes(bytes)).toBe(expected);
	});

	it('honours the decimals argument', () => {
		expect(formatBytes(172_150_000, 1)).toBe('164.2 MB');
	});
});

describe('formatUptime', () => {
	it.each([
		[null, '—'],
		[undefined, '—'],
		[100, '100%'],
		[100.4, '100%'],
		[99.999, '99.99%'],
		[99.5, '99.50%'],
		[0, '0.00%']
	])('%s -> %s', (percent, expected) => {
		expect(formatUptime(percent as number | null | undefined)).toBe(expected);
	});
});

function health(status: string, checks: Record<string, unknown>): HealthStatus {
	return { status, checks } as unknown as HealthStatus;
}

describe('describeHealth', () => {
	it('reports connecting before the first answer', () => {
		expect(describeHealth(null, false).label).toBe('CONNECTING');
	});

	it('reports offline when the server cannot be reached', () => {
		expect(describeHealth(null, true).label).toBe('OFFLINE');
	});

	it('reports healthy', () => {
		const result = describeHealth(health('healthy', { disk: { used_percent: '40%' } }), false);
		expect(result.label).toBe('SYSTEM ONLINE');
		expect(result.tip).toContain('40%');
	});

	it('names the missing internet connection first', () => {
		const result = describeHealth(
			health('degraded', { internet: { connected: false }, disk: { used_percent: '95%' }, database: false }),
			false
		);
		expect(result.label).toBe('NO INTERNET');
	});

	it('names a database outage', () => {
		const result = describeHealth(health('degraded', { internet: { connected: true }, database: false }), false);
		expect(result.label).toBe('DATABASE DOWN');
	});

	it('names a nearly full disk', () => {
		const result = describeHealth(
			health('degraded', { internet: { connected: true }, database: true, disk: { used_percent: '93.5%' } }),
			false
		);
		expect(result.label).toBe('DISK ALMOST FULL');
	});

	it('does not flag a disk at 90% or below', () => {
		const result = describeHealth(
			health('degraded', { internet: { connected: true }, database: true, disk: { used_percent: '90%' } }),
			false
		);
		expect(result.label).toBe('DEGRADED');
	});
});

describe('getRemainingTime', () => {
	const now = Date.parse('2026-01-01T12:00:00Z');
	const repo = (overrides: Partial<Repository>) =>
		({
			auto_patrol: 1,
			status: 'synced',
			interval_minutes: 60,
			last_sync: '2026-01-01T11:30:00Z',
			...overrides
		}) as Repository;

	it('counts down to the next sync', () => {
		expect(getRemainingTime(repo({}), true, now)).toBe('Next sync in: 30m');
		expect(getRemainingTime(repo({}), false, now)).toBe('30m');
	});

	it('shows seconds when under a minute', () => {
		expect(getRemainingTime(repo({ last_sync: '2026-01-01T11:00:30Z' }), false, now)).toBe('30s');
	});

	it('is manual-only without auto patrol', () => {
		expect(getRemainingTime(repo({ auto_patrol: 0 }), true, now)).toBe('Manual Patrol Only');
	});

	it('shows syncing when overdue, running or never synced', () => {
		expect(getRemainingTime(repo({ last_sync: '2026-01-01T09:00:00Z' }), true, now)).toBe('Syncing...');
		expect(getRemainingTime(repo({ status: 'syncing' }), true, now)).toBe('Syncing...');
		expect(getRemainingTime(repo({ last_sync: '' }), true, now)).toBe('Syncing...');
	});
});
