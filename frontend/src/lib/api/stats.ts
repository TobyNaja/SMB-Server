import { get } from './client';
import type { AuditEntry } from './audit';

export interface Stats {
	shares_count: number;
	users_count: number;
	groups_count: number;
	recent_audit: AuditEntry[];
}

export interface SambaStatus {
	smbd: boolean;
	nmbd: boolean;
	winbindd: boolean;
}

export const statsApi = {
	get: () => get<Stats>('/api/stats'),
	sambaStatus: () => get<SambaStatus>('/api/samba/status'),
};
