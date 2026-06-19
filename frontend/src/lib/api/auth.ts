import { get, post, del } from './client';

export interface LoginResponse {
	access_token: string;
	token_type: string;
	expires_in: number;
}

export interface MeResponse {
	username: string;
	is_admin: boolean;
	expires_at: string;
}

export interface AdminInfo {
	username: string;
	created_at: string;
	last_login?: string;
}

export const authApi = {
	login: (username: string, password: string) =>
		post<LoginResponse>('/auth/login', { username, password }),

	logout: () => post<{ message: string }>('/auth/logout'),

	me: () => get<MeResponse>('/auth/me'),

	changePassword: (old_password: string, new_password: string) =>
		post<{ message: string }>('/auth/change-password', { old_password, new_password }),

	listAdmins: () => get<{ admins: AdminInfo[]; count: number }>('/auth/admins'),

	addAdmin: (username: string, password: string) =>
		post<{ message: string }>('/auth/admins', { username, password }),

	deleteAdmin: (username: string) => del<{ message: string }>(`/auth/admins/${username}`)
};
