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

// One line of `getfacl` output: a POSIX ACL entry on a subfolder.
export interface SubfolderAclEntry {
	type: 'user' | 'group';
	name: string;    // empty = owner / owning-group entry (not editable)
	perms: string;   // e.g. "rwx", "r-x"
	default: boolean; // inheritance entry (applies to newly created children)
}

// Ordered subset of rwx that the backend accepts; "none" removes the entry.
export type SubfolderPerm = 'r' | 'rx' | 'rwx' | 'none';

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

	// Subfolder (POSIX ACL) permissions. `path` is relative to the share root;
	// empty/"." targets the root itself.
	getSubfolderPermissions: (name: string, path: string) =>
		get<{ share: string; path: string; entries: SubfolderAclEntry[] | null; locked: boolean }>(
			`/api/shares/${name}/subfolders/permissions?path=${encodeURIComponent(path || '.')}`
		),

	setSubfolderPermission: (
		name: string,
		data: { subfolder_path: string; username: string; permissions: SubfolderPerm; recursive?: boolean }
	) => post<{ message: string }>(`/api/shares/${name}/subfolders/permissions`, data),

	// Make a subfolder private to exactly `users` (empty = owner only), shutting
	// everyone else out and hiding it from their view.
	lockSubfolder: (
		name: string,
		data: { subfolder_path: string; users: string[]; permissions?: SubfolderPerm; recursive?: boolean }
	) => post<{ message: string }>(`/api/shares/${name}/subfolders/lock`, data),

	// Reopen a locked subfolder to the share's valid users.
	unlockSubfolder: (name: string, data: { subfolder_path: string; recursive?: boolean }) =>
		post<{ message: string }>(`/api/shares/${name}/subfolders/unlock`, data),

	getGlobal: () => get<GlobalSettings>('/api/shares/global')
};
