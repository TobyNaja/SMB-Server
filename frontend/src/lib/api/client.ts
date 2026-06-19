import { browser } from '$app/environment';
import { clearAuth, getToken } from '$lib/stores/auth.svelte';

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
	const token = getToken();
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...(options.headers as Record<string, string>)
	};
	if (token) {
		headers['Authorization'] = `Bearer ${token}`;
	}

	const res = await fetch(path, { ...options, headers, credentials: 'include' });

	if (res.status === 401) {
		clearAuth();
		if (browser) window.location.href = '/login';
		throw new Error('Unauthorized');
	}

	if (!res.ok) {
		const err = await res.json().catch(() => ({ detail: res.statusText }));
		throw new Error(err.detail || 'Request failed');
	}

	if (res.status === 204) return undefined as T;
	return res.json() as Promise<T>;
}

export const get = <T>(path: string) => request<T>(path);
export const post = <T>(path: string, body?: unknown) =>
	request<T>(path, { method: 'POST', body: JSON.stringify(body) });
export const patch = <T>(path: string, body?: unknown) =>
	request<T>(path, { method: 'PATCH', body: JSON.stringify(body) });
export const del = <T>(path: string) => request<T>(path, { method: 'DELETE' });
