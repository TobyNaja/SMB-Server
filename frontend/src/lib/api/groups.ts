import { get, post, del } from './client';

export const groupsApi = {
	list: () => get<{ groups: string[] }>('/api/groups'),

	create: (group_name: string) => post<{ message: string }>('/api/groups', { group_name }),

	addMember: (group: string, username: string) =>
		post<{ message: string }>(`/api/groups/${group}/members/${username}`),

	removeMember: (group: string, username: string) =>
		del<{ message: string }>(`/api/groups/${group}/members/${username}`)
};
