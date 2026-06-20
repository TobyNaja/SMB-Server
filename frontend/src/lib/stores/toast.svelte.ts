export type ToastType = 'success' | 'error' | 'info';

export interface Toast {
	id: string;
	type: ToastType;
	message: string;
}

let _toasts = $state<Toast[]>([]);

function generateId(): string {
	if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
		return crypto.randomUUID();
	}
	// Fallback for non-secure contexts (HTTP + IP)
	return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
		const r = (Math.random() * 16) | 0;
		return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
	});
}

export function getToasts(): Toast[] {
	return _toasts;
}

export function addToast(message: string, type: ToastType = 'success'): void {
	const id = generateId();
	_toasts = [..._toasts, { id, type, message }];
	setTimeout(() => {
		_toasts = _toasts.filter((t) => t.id !== id);
	}, 4000);
}

export function toast(message: string): void {
	addToast(message, 'success');
}

export function toastError(message: string): void {
	addToast(message, 'error');
}
