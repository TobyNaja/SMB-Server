import { browser } from '$app/environment';

export interface CurrentUser {
	username: string;
	is_admin: boolean;
	expires_at: string;
}

let _user = $state<CurrentUser | null>(null);
let _token = $state<string | null>(browser ? localStorage.getItem('access_token') : null);

export function getUser(): CurrentUser | null {
	return _user;
}

export function getToken(): string | null {
	return _token;
}

export function isAuthenticated(): boolean {
	return _token !== null;
}

export function setAuth(user: CurrentUser, tok: string): void {
	_user = user;
	_token = tok;
	if (browser) localStorage.setItem('access_token', tok);
}

export function clearAuth(): void {
	_user = null;
	_token = null;
	if (browser) localStorage.removeItem('access_token');
}
