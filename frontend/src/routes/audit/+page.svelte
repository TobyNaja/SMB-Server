<script lang="ts">
	import { onMount } from 'svelte';
	import { auditApi, type AuditEntry } from '$lib/api/audit';
	import { toastError } from '$lib/stores/toast.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import { Download, RefreshCw } from 'lucide-svelte';

	let logs         = $state<AuditEntry[]>([]);
	let loading      = $state(true);
	let limit        = $state(500);
	let filterAction = $state('');
	let filterActor  = $state('');

	// Pagination
	let page     = $state(1);
	let pageSize = $state(25);

	const paged = $derived(logs.slice((page - 1) * pageSize, page * pageSize));

	async function load() {
		loading = true;
		page = 1;
		try {
			const r = await auditApi.getLogs(limit, filterAction || undefined, filterActor || undefined);
			logs = r.logs ?? [];
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to load audit log');
		} finally {
			loading = false;
		}
	}

	function statusColor(status: string) {
		if (status === 'success') return 'bg-green-100 text-gcp-green';
		if (status === 'failure') return 'bg-red-100 text-gcp-red';
		return 'bg-gray-100 text-gcp-muted';
	}

	function formatTime(ts: string) {
		return new Date(ts).toLocaleString('th-TH', { dateStyle: 'short', timeStyle: 'medium' });
	}

	function exportCSV() {
		const header = ['Time', 'Action', 'Actor', 'Resource Type', 'Resource', 'Status', 'IP', 'Details'];
		const rows = logs.map(e => [
			e.timestamp, e.action, e.actor,
			e.resource_type ?? '', e.resource_name ?? '',
			e.status, e.client_ip ?? '',
			e.details ? JSON.stringify(e.details) : '',
		]);
		const csv = [header, ...rows]
			.map(r => r.map(cell => `"${String(cell).replace(/"/g, '""')}"`).join(','))
			.join('\n');
		const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
		const url  = URL.createObjectURL(blob);
		const a    = document.createElement('a');
		a.href = url;
		a.download = `audit_${new Date().toISOString().slice(0, 10)}.csv`;
		a.click();
		URL.revokeObjectURL(url);
	}

	onMount(load);
</script>

<div>
	<!-- Toolbar -->
	<div class="mb-4 flex flex-wrap items-end gap-2">
		<h1 class="page-title mr-2">Audit Log</h1>
		<input bind:value={filterAction} placeholder="Filter action…" class="input-field w-40 text-xs" />
		<input bind:value={filterActor}  placeholder="Filter actor…"  class="input-field w-32 text-xs" />
		<select bind:value={limit} class="select-field w-28 text-xs">
			<option value={100}>Load 100</option>
			<option value={250}>Load 250</option>
			<option value={500}>Load 500</option>
		</select>
		<button onclick={load}
			class="flex items-center gap-1.5 rounded border border-gcp-border bg-white px-3 py-2
				text-xs text-gcp-dark hover:bg-gcp-bg transition-colors">
			<RefreshCw size={12} />Refresh
		</button>
		<button onclick={exportCSV} disabled={logs.length === 0}
			class="flex items-center gap-1.5 rounded border border-gcp-border bg-white px-3 py-2
				text-xs text-gcp-dark hover:bg-gcp-bg disabled:opacity-50 transition-colors">
			<Download size={12} />Export CSV
		</button>
	</div>

	{#if loading}
		<div class="text-sm text-gcp-muted">Loading…</div>
	{:else if logs.length === 0}
		<div class="text-sm text-gcp-muted">No audit entries found</div>
	{:else}
		<div class="card overflow-hidden">
			<table class="w-full text-xs">
				<thead>
					<tr class="border-b border-gcp-border text-left font-medium uppercase text-gcp-muted">
						<th class="px-4 py-3">Time</th>
						<th class="px-4 py-3">Action</th>
						<th class="px-4 py-3">Actor</th>
						<th class="px-4 py-3">Resource</th>
						<th class="px-4 py-3">Status</th>
						<th class="px-4 py-3">Details</th>
						<th class="px-4 py-3">IP</th>
					</tr>
				</thead>
				<tbody>
					{#each paged as entry}
						{@const detailStr = entry.details && Object.keys(entry.details).length > 0 ? JSON.stringify(entry.details) : ''}
						<tr class="border-b border-gcp-border/50 hover:bg-gcp-bg">
							<td class="whitespace-nowrap px-4 py-2.5 text-gcp-muted">{formatTime(entry.timestamp)}</td>
							<td class="px-4 py-2.5 font-mono font-medium text-gcp-dark">{entry.action}</td>
							<td class="px-4 py-2.5 text-gcp-blue">{entry.actor}</td>
							<td class="px-4 py-2.5">
								{#if entry.resource_name}
									<span class="text-gcp-muted">
										{entry.resource_type}: <span class="font-medium text-gcp-dark">{entry.resource_name}</span>
									</span>
								{:else}
									<span class="text-gcp-muted">—</span>
								{/if}
							</td>
							<td class="px-4 py-2.5">
								<span class="badge {statusColor(entry.status)}">{entry.status}</span>
							</td>
							<td class="max-w-xs truncate px-4 py-2.5 font-mono text-gcp-muted" title={detailStr}>{detailStr || '—'}</td>
							<td class="px-4 py-2.5 font-mono text-gcp-muted">{entry.client_ip || '—'}</td>
						</tr>
					{/each}
				</tbody>
			</table>

			<div class="border-t border-gcp-border px-4 pb-3">
				<Pagination
					total={logs.length}
					{page}
					{pageSize}
					pageSizeOptions={[10, 25, 50, 100]}
					onPageChange={(p) => (page = p)}
					onPageSizeChange={(s) => { pageSize = s; page = 1; }}
				/>
			</div>
		</div>
	{/if}
</div>
