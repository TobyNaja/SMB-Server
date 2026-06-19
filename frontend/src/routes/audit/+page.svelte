<script lang="ts">
	import { onMount } from 'svelte';
	import { auditApi, type AuditEntry } from '$lib/api/audit';
	import { toastError } from '$lib/stores/toast.svelte';

	let logs = $state<AuditEntry[]>([]);
	let loading = $state(true);

	let limit = $state(100);
	let filterAction = $state('');
	let filterActor = $state('');

	async function load() {
		loading = true;
		try {
			const r = await auditApi.getLogs(limit, filterAction || undefined, filterActor || undefined);
			logs = r.logs;
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to load audit log');
		} finally {
			loading = false;
		}
	}

	function statusColor(status: string) {
		if (status === 'success') return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400';
		if (status === 'failure') return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400';
		return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400';
	}

	function formatTime(ts: string) {
		return new Date(ts).toLocaleString('th-TH', { dateStyle: 'short', timeStyle: 'medium' });
	}

	onMount(load);
</script>

<div>
	<div class="mb-6 flex flex-wrap items-end gap-3">
		<h1 class="flex-none text-lg font-semibold text-gray-800 dark:text-white">Audit Log</h1>

		<div class="flex flex-1 flex-wrap gap-2">
			<input bind:value={filterAction} placeholder="Filter action…"
				class="input-field w-44 text-xs" />
			<input bind:value={filterActor} placeholder="Filter actor…"
				class="input-field w-36 text-xs" />
			<select bind:value={limit} class="select-field w-28 text-xs">
				<option value={50}>Last 50</option>
				<option value={100}>Last 100</option>
				<option value={500}>Last 500</option>
				<option value={1000}>Last 1000</option>
			</select>
			<button onclick={load}
				class="rounded bg-blue-600 px-3 py-2 text-xs text-white hover:bg-blue-700">
				Refresh
			</button>
		</div>
	</div>

	{#if loading}
		<div class="text-sm text-gray-400">Loading…</div>
	{:else if logs.length === 0}
		<div class="text-sm text-gray-400">No audit entries found</div>
	{:else}
		<div class="rounded-xl bg-white shadow-sm dark:bg-gray-800 overflow-hidden">
			<table class="w-full text-xs">
				<thead>
					<tr class="border-b border-gray-100 text-left font-medium uppercase text-gray-400 dark:border-gray-700">
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
					{#each logs as entry}
						{@const detailStr = entry.details && Object.keys(entry.details).length > 0 ? JSON.stringify(entry.details) : ''}
						<tr class="border-b border-gray-50 hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-700/50">
							<td class="whitespace-nowrap px-4 py-2.5 text-gray-500">{formatTime(entry.timestamp)}</td>
							<td class="px-4 py-2.5 font-mono font-medium text-gray-800 dark:text-white">{entry.action}</td>
							<td class="px-4 py-2.5 text-blue-600 dark:text-blue-400">{entry.actor}</td>
							<td class="px-4 py-2.5">
								{#if entry.resource_name}
									<span class="text-gray-600 dark:text-gray-400">
										{entry.resource_type}: <span class="font-medium">{entry.resource_name}</span>
									</span>
								{:else}
									<span class="text-gray-400">—</span>
								{/if}
							</td>
							<td class="px-4 py-2.5">
								<span class="rounded-full px-2 py-0.5 {statusColor(entry.status)}">{entry.status}</span>
							</td>
							<td class="max-w-xs truncate px-4 py-2.5 text-gray-500" title={detailStr}>{detailStr || '—'}</td>
							<td class="px-4 py-2.5 font-mono text-gray-400">{entry.client_ip || '—'}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<p class="mt-2 text-right text-xs text-gray-400">{logs.length} entries</p>
	{/if}
</div>

