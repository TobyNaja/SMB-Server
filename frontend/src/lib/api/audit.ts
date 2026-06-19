import { get } from './client';

export interface AuditEntry {
	timestamp: string;
	action: string;
	actor: string;
	resource_type: string;
	resource_name: string;
	status: string;
	details: Record<string, unknown>;
	client_ip?: string;
}

export const auditApi = {
	getLogs: (limit = 100, action = '', actor = '') =>
		get<{ logs: AuditEntry[]; count: number }>(
			`/api/audit/logs?limit=${limit}&action=${encodeURIComponent(action)}&actor=${encodeURIComponent(actor)}`
		)
};
