import { get, post, del } from './client';

export interface BuiltinGroup {
	name: string;
	full_name: string;
	description: string;
	color: string;
	icon: string;
	members: string[];
	member_count: number;
}

export const builtinApi = {
	list: () => get<{ groups: BuiltinGroup[] }>('/api/builtin'),

	getMembers: (group: string) => get<BuiltinGroup>(`/api/builtin/${group}/members`),

	addMember: (group: string, username: string) =>
		post<{ message: string; members: string[] }>(`/api/builtin/${group}/members`, { username }),

	removeMember: (group: string, username: string) =>
		del<{ message: string; members: string[] }>(`/api/builtin/${group}/members/${username}`)
};
