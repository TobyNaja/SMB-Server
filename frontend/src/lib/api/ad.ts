import { get } from './client';

export interface ADUser {
	username: string;
	display_name: string;
	email: string;
	department: string;
	title: string;
	ou: string;
	source: 'ad';
}

export interface ADGroup {
	name: string;
	description: string;
	ou: string;
	smb_name: string;
	source: 'ad';
}

export interface ADStatus {
	ldap_server: string;
	domain: string;
	base_dn: string;
	bind_dn: string;
	connected: boolean;
	error?: string;
}

export interface OU {
	name: string;
	dn: string;
	description: string;
}

export const adApi = {
	status: () => get<ADStatus>('/api/ad/status'),

	searchUsers: (q = '', ou = '', limit = 50) =>
		get<{ users: ADUser[]; count: number; domain: string }>(
			`/api/ad/users?q=${encodeURIComponent(q)}&ou=${encodeURIComponent(ou)}&limit=${limit}`
		),

	getUser: (username: string) => get<ADUser>(`/api/ad/users/${encodeURIComponent(username)}`),

	searchGroups: (q = '', limit = 50) =>
		get<{ groups: ADGroup[]; count: number; domain: string }>(
			`/api/ad/groups?q=${encodeURIComponent(q)}&limit=${limit}`
		),

	listOUs: () => get<{ ous: OU[]; domain: string }>('/api/ad/ous')
};
