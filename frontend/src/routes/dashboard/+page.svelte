<script lang="ts">
	import { onMount } from 'svelte';
	import { statsApi, type Stats, type SambaStatus } from '$lib/api/stats';
	import { adApi, type ADStatus } from '$lib/api/ad';
	import { toastError } from '$lib/stores/toast.svelte';
	import { Folder, User, Users, CheckCircle2, XCircle, RefreshCw } from 'lucide-svelte';

	let stats     = $state<Stats | null>(null);
	let samba     = $state<SambaStatus | null>(null);
	let adStatus  = $state<ADStatus | null>(null);
	let loading   = $state(true);
	let refreshing = $state(false);

	async function load() {
		try {
			// allSettled: one slow/failing endpoint won't block the whole dashboard
			const [statsRes, sambaRes, adRes] = await Promise.allSettled([
				statsApi.get(),
				statsApi.sambaStatus(),
				adApi.status(),
			]);
			if (statsRes.status === 'fulfilled') stats = statsRes.value;
			if (sambaRes.status === 'fulfilled') samba = sambaRes.value;
			if (adRes.status   === 'fulfilled') adStatus = adRes.value;
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to load dashboard');
		} finally {
			loading = false;
			refreshing = false;
		}
	}

	function refresh() {
		refreshing = true;
		load();
	}

	function formatTime(ts: string) {
		return new Date(ts).toLocaleString('th-TH', { dateStyle: 'short', timeStyle: 'medium' });
	}

	function statusBadge(status: string) {
		if (status === 'success') return 'bg-green-100 text-green-800';
		if (status === 'failure') return 'bg-red-100 text-red-800';
		return 'bg-gray-100 text-gcp-muted';
	}

	onMount(load);
</script>

<div class="space-y-6">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<h1 class="page-title">Dashboard</h1>
		<button
			onclick={refresh}
			disabled={refreshing}
			class="flex items-center gap-1.5 rounded border border-gcp-border bg-white px-3 py-1.5
				text-xs text-gcp-muted hover:bg-gcp-bg disabled:opacity-60 transition-colors"
		>
			<RefreshCw size={12} class={refreshing ? 'animate-spin' : ''} />
			Refresh
		</button>
	</div>

	{#if loading}
		<div class="text-sm text-gcp-muted">Loading…</div>
	{:else}
		<!-- Stat cards -->
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
			<a href="/shares" class="card flex items-center gap-4 p-5 hover:shadow-sm transition-shadow">
				<div class="flex h-10 w-10 items-center justify-center rounded bg-gcp-blue-light">
					<Folder size={20} class="text-gcp-blue" />
				</div>
				<div>
					<div class="text-2xl font-semibold text-gcp-dark">{stats?.shares_count ?? 0}</div>
					<div class="text-xs text-gcp-muted">Shares</div>
				</div>
			</a>

			<a href="/users" class="card flex items-center gap-4 p-5 hover:shadow-sm transition-shadow">
				<div class="flex h-10 w-10 items-center justify-center rounded bg-green-50">
					<User size={20} class="text-gcp-green" />
				</div>
				<div>
					<div class="text-2xl font-semibold text-gcp-dark">{stats?.users_count ?? 0}</div>
					<div class="text-xs text-gcp-muted">Local Users</div>
				</div>
			</a>

			<a href="/groups" class="card flex items-center gap-4 p-5 hover:shadow-sm transition-shadow">
				<div class="flex h-10 w-10 items-center justify-center rounded bg-purple-50">
					<Users size={20} class="text-purple-600" />
				</div>
				<div>
					<div class="text-2xl font-semibold text-gcp-dark">{stats?.groups_count ?? 0}</div>
					<div class="text-xs text-gcp-muted">Groups</div>
				</div>
			</a>
		</div>

		<!-- Service status + AD status -->
		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			<!-- Samba service status -->
			<div class="card p-5">
				<h2 class="mb-4 text-xs font-semibold uppercase tracking-wide text-gcp-muted">Samba Services</h2>
				<div class="space-y-2.5">
					{#each [
						{ name: 'smbd',     label: 'File Server (smbd)',   running: samba?.smbd },
						{ name: 'nmbd',     label: 'NetBIOS (nmbd)',        running: samba?.nmbd },
						{ name: 'winbindd', label: 'WinBind (winbindd)',    running: samba?.winbindd },
					] as svc}
						<div class="flex items-center justify-between">
							<span class="text-sm text-gcp-dark">{svc.label}</span>
							<div class="flex items-center gap-1.5 text-xs">
								{#if svc.running}
									<CheckCircle2 size={14} class="text-gcp-green" />
									<span class="text-gcp-green font-medium">Running</span>
								{:else}
									<XCircle size={14} class="text-gcp-red" />
									<span class="text-gcp-red font-medium">Stopped</span>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			</div>

			<!-- AD status -->
			<div class="card p-5">
				<h2 class="mb-4 text-xs font-semibold uppercase tracking-wide text-gcp-muted">Active Directory</h2>
				{#if adStatus}
					<div class="flex items-center gap-2 mb-3">
						{#if adStatus.connected}
							<CheckCircle2 size={16} class="text-gcp-green flex-none" />
							<span class="text-sm font-medium text-gcp-green">Connected</span>
						{:else}
							<XCircle size={16} class="text-gcp-red flex-none" />
							<span class="text-sm font-medium text-gcp-red">Disconnected</span>
						{/if}
					</div>
					<div class="space-y-1 text-xs text-gcp-muted">
						<div>Domain: <span class="font-mono text-gcp-dark">{adStatus.domain}</span></div>
						<div>Server: <span class="font-mono text-gcp-dark">{adStatus.ldap_server}</span></div>
						{#if !adStatus.connected && adStatus.error}
							<div class="mt-2 text-gcp-red">{adStatus.error}</div>
						{/if}
					</div>
				{:else}
					<p class="text-sm text-gcp-muted">Unable to reach AD</p>
				{/if}
			</div>
		</div>

		<!-- Recent audit -->
		{#if stats?.recent_audit?.length}
			<div class="card overflow-hidden">
				<div class="border-b border-gcp-border px-5 py-3">
					<h2 class="text-xs font-semibold uppercase tracking-wide text-gcp-muted">Recent Activity</h2>
				</div>
				<table class="w-full text-xs">
					<thead>
						<tr class="border-b border-gcp-border text-left font-medium text-gcp-muted">
							<th class="px-5 py-2.5">Time</th>
							<th class="px-5 py-2.5">Action</th>
							<th class="px-5 py-2.5">Actor</th>
							<th class="px-5 py-2.5">Status</th>
						</tr>
					</thead>
					<tbody>
						{#each stats.recent_audit as entry}
							<tr class="border-b border-gcp-border/50 hover:bg-gcp-bg">
								<td class="px-5 py-2.5 text-gcp-muted">{formatTime(entry.timestamp)}</td>
								<td class="px-5 py-2.5 font-mono font-medium text-gcp-dark">{entry.action}</td>
								<td class="px-5 py-2.5 text-gcp-blue">{entry.actor}</td>
								<td class="px-5 py-2.5">
									<span class="badge {statusBadge(entry.status)}">{entry.status}</span>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
				<div class="px-5 py-2.5">
					<a href="/audit" class="text-xs text-gcp-blue hover:underline">View all audit logs →</a>
				</div>
			</div>
		{/if}
	{/if}
</div>
