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
	domain: string;
	connected: boolean;
}

export interface OU {
	name: string;
	dn: string;
	description: string;
}

export const adApi = {
	status: () => get<ADStatus>('/api/ad/status'),

	searchUsers: (q = '', ou = '') =>
		get<{ users: ADUser[]; count: number }>(
			`/api/ad/users?q=${encodeURIComponent(q)}&ou=${encodeURIComponent(ou)}`
		),

	getUser: (username: string) => get<ADUser>(`/api/ad/users/${encodeURIComponent(username)}`),

	searchGroups: (q = '') =>
		get<{ groups: ADGroup[]; count: number }>(
			`/api/ad/groups?q=${encodeURIComponent(q)}`
		),

	listOUs: () => get<{ ous: OU[] }>('/api/ad/ous')
};
