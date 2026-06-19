import { get, post, patch, del } from './client';

export interface Share {
	name: string;
	path: string;
	comment: string;
	browseable: boolean;
	read_only: boolean;
	guest_ok: boolean;
	abse: boolean;
	valid_users: string[];
	write_list: string[];
	read_list: string[];
	admin_users: string[];
	invalid_users: string[];
	create_mask: string;
	directory_mask: string;
}

export type PermissionType = 'valid_users' | 'write_list' | 'read_list' | 'admin_users' | 'invalid_users';

export interface GlobalSettings {
	workgroup: string;
	realm: string;
	security: string;
	netbios_name: string;
	server_string: string;
	abse: boolean;
}

export const sharesApi = {
	list: () => get<{ shares: Share[] }>('/api/shares'),

	get: (name: string) => get<Share>(`/api/shares/${name}`),

	create: (data: {
		name: string;
		path: string;
		comment?: string;
		browseable?: boolean;
		guest_ok?: boolean;
		abse?: boolean;
	}) => post<{ message: string }>('/api/shares', data),

	update: (
		name: string,
		data: Partial<{
			comment: string;
			browseable: boolean;
			guest_ok: boolean;
			read_only: boolean;
			abse: boolean;
			create_mask: string;
			directory_mask: string;
		}>
	) => patch<{ message: string }>(`/api/shares/${name}`, data),

	delete: (name: string) => del<{ message: string }>(`/api/shares/${name}`),

	setPermissions: (name: string, permission_type: PermissionType, users: string[]) =>
		post<{ message: string }>(`/api/shares/${name}/permissions`, { permission_type, users }),

	toggleABSE: (name: string, enabled: boolean) =>
		patch<{ message: string }>(`/api/shares/${name}/abse?enabled=${enabled}`),

	getGlobal: () => get<GlobalSettings>('/api/shares/global')
};
