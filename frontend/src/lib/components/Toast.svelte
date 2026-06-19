<script lang="ts">
	import { getToasts } from '$lib/stores/toast.svelte';
</script>

{#if getToasts().length > 0}
	<div class="fixed top-4 right-4 z-50 flex flex-col gap-2">
		{#each getToasts() as t (t.id)}
			<div
				class="flex items-center gap-2 rounded-lg px-4 py-3 text-sm font-medium shadow-lg transition-all {t.type === 'error'
					? 'bg-red-600 text-white'
					: t.type === 'info'
						? 'bg-blue-600 text-white'
						: 'bg-green-600 text-white'}"
			>
				{#if t.type === 'error'}
					<span>✕</span>
				{:else}
					<span>✓</span>
				{/if}
				{t.message}
			</div>
		{/each}
	</div>
{/if}
