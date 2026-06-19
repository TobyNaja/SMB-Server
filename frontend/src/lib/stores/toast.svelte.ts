export type ToastType = 'success' | 'error' | 'info';

export interface Toast {
	id: string;
	type: ToastType;
	message: string;
}

let _toasts = $state<Toast[]>([]);

export function getToasts(): Toast[] {
	return _toasts;
}

export function addToast(message: string, type: ToastType = 'success'): void {
	const id = crypto.randomUUID();
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
