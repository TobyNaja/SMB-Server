export interface CurrentUser {
	username: string;
	is_admin: boolean;
	expires_at: string;
}

let _user = $state<CurrentUser | null>(null);

export function getUser(): CurrentUser | null {
	return _user;
}

export function isAuthenticated(): boolean {
	return _user !== null;
}

export function setAuth(user: CurrentUser): void {
	_user = user;
}

export function clearAuth(): void {
	_user = null;
}
