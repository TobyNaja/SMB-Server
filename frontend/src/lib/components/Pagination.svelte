<script lang="ts">
	import { ChevronLeft, ChevronRight } from 'lucide-svelte';

	interface Props {
		total: number;
		page: number;
		pageSize: number;
		pageSizeOptions?: number[];
		onPageChange: (p: number) => void;
		onPageSizeChange?: (size: number) => void;
	}

	let {
		total,
		page,
		pageSize,
		pageSizeOptions,
		onPageChange,
		onPageSizeChange,
	}: Props = $props();

	const totalPages = $derived(Math.ceil(total / pageSize));
	const start      = $derived(total === 0 ? 0 : (page - 1) * pageSize + 1);
	const end        = $derived(Math.min(page * pageSize, total));

	function pageNums(): (number | '…')[] {
		if (totalPages <= 7) return Array.from({ length: totalPages }, (_, i) => i + 1);
		const out: (number | '…')[] = [1];
		if (page > 3) out.push('…');
		for (let i = Math.max(2, page - 1); i <= Math.min(totalPages - 1, page + 1); i++) {
			out.push(i);
		}
		if (page < totalPages - 2) out.push('…');
		out.push(totalPages);
		return out;
	}
</script>

<div class="flex flex-wrap items-center justify-between gap-3 pt-3">
	<!-- Row info + per-page -->
	<div class="flex items-center gap-3 text-xs text-gcp-muted">
		{#if total > 0}
			<span>{start}–{end} of {total}</span>
		{:else}
			<span>0 items</span>
		{/if}
		{#if pageSizeOptions && onPageSizeChange}
			<select
				value={pageSize}
				onchange={(e) => { onPageSizeChange(parseInt((e.currentTarget as HTMLSelectElement).value, 10)); onPageChange(1); }}
				class="rounded border border-gcp-border bg-white px-2 py-1 text-xs text-gcp-dark
					focus:border-gcp-blue focus:outline-none"
			>
				{#each pageSizeOptions as o}
					<option value={o}>{o} / page</option>
				{/each}
			</select>
		{/if}
	</div>

	<!-- Page buttons -->
	{#if totalPages > 1}
		<div class="flex items-center gap-1">
			<button
				onclick={() => onPageChange(page - 1)}
				disabled={page === 1}
				class="flex h-7 w-7 items-center justify-center rounded border border-gcp-border
					text-gcp-muted hover:bg-gcp-bg disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
				aria-label="Previous page"
			>
				<ChevronLeft size={13} />
			</button>

			{#each pageNums() as p}
				{#if p === '…'}
					<span class="flex h-7 w-7 items-center justify-center text-xs text-gcp-muted">…</span>
				{:else}
					<button
						onclick={() => onPageChange(p as number)}
						class="flex h-7 min-w-7 items-center justify-center rounded border px-1.5 text-xs
							transition-colors
							{page === p
								? 'border-gcp-blue bg-gcp-blue text-white'
								: 'border-gcp-border text-gcp-dark hover:bg-gcp-bg'}"
					>
						{p}
					</button>
				{/if}
			{/each}

			<button
				onclick={() => onPageChange(page + 1)}
				disabled={page === totalPages}
				class="flex h-7 w-7 items-center justify-center rounded border border-gcp-border
					text-gcp-muted hover:bg-gcp-bg disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
				aria-label="Next page"
			>
				<ChevronRight size={13} />
			</button>
		</div>
	{/if}
</div>
