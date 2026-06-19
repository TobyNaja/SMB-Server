import { get, post, del } from './client';

export interface User {
	username: string;
	uid: string;
	fullname: string;
	disabled: boolean;
}

export const usersApi = {
	list: () => get<{ users: User[] }>('/api/users'),

	create: (username: string, password: string, fullname = '') =>
		post<{ message: string }>('/api/users', { username, password, fullname }),

	delete: (username: string) => del<{ message: string }>(`/api/users/${username}`),

	setPassword: (username: string, password: string) =>
		post<{ message: string }>(`/api/users/${username}/password`, { password })
};
