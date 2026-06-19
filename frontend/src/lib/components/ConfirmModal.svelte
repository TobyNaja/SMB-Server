<script lang="ts">
	import { AlertTriangle } from 'lucide-svelte';

	interface Props {
		open: boolean;
		title?: string;
		message: string;
		confirmLabel?: string;
		danger?: boolean;
		onconfirm: () => void;
		oncancel: () => void;
	}

	let {
		open,
		title = 'Confirm',
		message,
		confirmLabel = 'Confirm',
		danger = false,
		onconfirm,
		oncancel,
	}: Props = $props();
</script>

{#if open}
	<!-- Backdrop -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/30"
		onclick={(e) => { if (e.target === e.currentTarget) oncancel(); }}
	>
		<div class="card w-full max-w-sm p-6 shadow-lg">
			<div class="mb-4 flex items-start gap-3">
				{#if danger}
					<AlertTriangle size={20} class="mt-0.5 flex-none text-gcp-red" />
				{/if}
				<div>
					<h3 class="text-sm font-semibold text-gcp-dark">{title}</h3>
					<p class="mt-1 text-sm text-gcp-muted">{message}</p>
				</div>
			</div>
			<div class="flex justify-end gap-2">
				<button onclick={oncancel} class="btn-secondary text-xs px-3 py-1.5">Cancel</button>
				<button
					onclick={onconfirm}
					class="{danger ? 'btn-danger' : 'btn-primary'} text-xs px-3 py-1.5"
				>{confirmLabel}</button>
			</div>
		</div>
	</div>
{/if}
