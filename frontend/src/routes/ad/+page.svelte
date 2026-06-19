<script lang="ts">
	import { onMount } from 'svelte';
	import { adApi, type ADUser, type ADGroup, type ADStatus } from '$lib/api/ad';
	import { sharesApi, type Share, type PermissionType } from '$lib/api/shares';
	import { toast, toastError } from '$lib/stores/toast.svelte';
	import { CheckCircle2, XCircle, Plus } from 'lucide-svelte';
	import Pagination from '$lib/components/Pagination.svelte';

	let status = $state<ADStatus | null>(null);
	let users  = $state<ADUser[]>([]);
	let groups = $state<ADGroup[]>([]);
	let shares = $state<Share[]>([]);
	let loading = $state(false);
	let tab = $state<'users' | 'groups'>('users');
	let userQuery  = $state('');
	let groupQuery = $state('');

	// Pagination
	const PAGE_SIZE = 20;
	let userPage  = $state(1);
	let groupPage = $state(1);

	const pagedUsers  = $derived(users.slice((userPage - 1) * PAGE_SIZE, userPage * PAGE_SIZE));
	const pagedGroups = $derived(groups.slice((groupPage - 1) * PAGE_SIZE, groupPage * PAGE_SIZE));

	// Quick-add to share dialog
	let qaOpen     = $state(false);
	let qaName     = $state('');  // username or smb_name
	let qaShare    = $state('');
	let qaPerm     = $state<PermissionType>('write_list');
	let qaAdding   = $state(false);

	const permOptions: { value: PermissionType; label: string }[] = [
		{ value: 'write_list',    label: 'Write List' },
		{ value: 'read_list',     label: 'Read List' },
		{ value: 'valid_users',   label: 'Valid Users' },
		{ value: 'admin_users',   label: 'Admin Users' },
		{ value: 'invalid_users', label: 'Blocked' },
	];

	onMount(async () => {
		const [adRes, sharesRes] = await Promise.allSettled([adApi.status(), sharesApi.list()]);
		if (adRes.status     === 'fulfilled') status = adRes.value;
		if (sharesRes.status === 'fulfilled') shares = sharesRes.value.shares ?? [];
	});

	async function searchUsers() {
		loading = true;
		try {
			const r = await adApi.searchUsers(userQuery);
			users = r.users ?? [];
			userPage = 1;
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'AD search failed');
		} finally {
			loading = false;
		}
	}

	async function searchGroups() {
		loading = true;
		try {
			const r = await adApi.searchGroups(groupQuery);
			groups = r.groups ?? [];
			groupPage = 1;
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'AD search failed');
		} finally {
			loading = false;
		}
	}

	function openQuickAdd(name: string) {
		qaName = name;
		qaShare = shares[0]?.name ?? '';
		qaPerm = 'write_list';
		qaOpen = true;
	}

	async function quickAdd() {
		if (!qaShare || !qaName) return;
		qaAdding = true;
		try {
			// Get current list, append, set
			const share = await sharesApi.get(qaShare);
			const existing: Record<PermissionType, string[]> = {
				write_list:    share.write_list,
				read_list:     share.read_list,
				valid_users:   share.valid_users,
				admin_users:   share.admin_users,
				invalid_users: share.invalid_users,
			};
			const newList = [...new Set([...(existing[qaPerm] ?? []), qaName])];
			await sharesApi.setPermissions(qaShare, qaPerm, newList);
			toast(`Added '${qaName}' to ${qaShare} (${qaPerm.replace('_', ' ')})`);
			qaOpen = false;
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to add to share');
		} finally {
			qaAdding = false;
		}
	}
</script>

<!-- Quick-add modal -->
{#if qaOpen}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/30"
		onclick={(e) => { if (e.target === e.currentTarget) qaOpen = false; }}>
		<div class="card w-80 p-5 shadow-lg">
			<h3 class="mb-4 text-sm font-semibold text-gcp-dark">Add to share</h3>
			<p class="mb-3 text-xs text-gcp-muted">
				Adding: <span class="font-mono font-medium text-gcp-dark">{qaName}</span>
			</p>
			<div class="space-y-3">
				<div>
					<label for="qa-share" class="mb-1 block text-xs text-gcp-muted">Share</label>
					<select id="qa-share" bind:value={qaShare} class="select-field w-full">
						{#each shares as s}<option value={s.name}>{s.name}</option>{/each}
					</select>
				</div>
				<div>
					<label for="qa-perm" class="mb-1 block text-xs text-gcp-muted">Permission list</label>
					<select id="qa-perm" bind:value={qaPerm} class="select-field w-full">
						{#each permOptions as o}<option value={o.value}>{o.label}</option>{/each}
					</select>
				</div>
			</div>
			<div class="mt-4 flex justify-end gap-2">
				<button onclick={() => (qaOpen = false)} class="btn-secondary text-xs px-3 py-1.5">Cancel</button>
				<button onclick={quickAdd} disabled={qaAdding || !qaShare}
					class="btn-primary text-xs px-3 py-1.5">
					{qaAdding ? 'Adding…' : 'Add'}
				</button>
			</div>
		</div>
	</div>
{/if}

<div>
	<h1 class="page-title mb-4">Active Directory</h1>

	<!-- Status bar -->
	{#if status}
		<div class="card mb-5 flex items-center gap-3 p-4">
			{#if status.connected}
				<CheckCircle2 size={16} class="flex-none text-gcp-green" />
				<span class="text-sm font-medium text-gcp-green">Connected</span>
			{:else}
				<XCircle size={16} class="flex-none text-gcp-red" />
				<span class="text-sm font-medium text-gcp-red">Disconnected</span>
			{/if}
			<span class="text-sm font-medium text-gcp-dark">{status.domain}</span>
			<span class="text-xs text-gcp-muted">{status.ldap_server}</span>
			{#if !status.connected && status.error}
				<span class="ml-auto text-xs text-gcp-red">{status.error}</span>
			{/if}
		</div>
	{/if}

	<!-- Tabs -->
	<div class="mb-4 flex border-b border-gcp-border">
		<button onclick={() => (tab = 'users')}
			class="px-4 py-2 text-sm font-medium transition-colors
				{tab === 'users' ? 'border-b-2 border-gcp-blue text-gcp-blue' : 'text-gcp-muted hover:text-gcp-dark'}">
			Users
		</button>
		<button onclick={() => (tab = 'groups')}
			class="px-4 py-2 text-sm font-medium transition-colors
				{tab === 'groups' ? 'border-b-2 border-gcp-blue text-gcp-blue' : 'text-gcp-muted hover:text-gcp-dark'}">
			Groups
		</button>
	</div>

	{#if tab === 'users'}
		<form onsubmit={(e) => { e.preventDefault(); searchUsers(); }} class="mb-4 flex gap-2">
			<input bind:value={userQuery} placeholder="Search by username, name, or email…"
				class="input-field flex-1" />
			<button type="submit" class="btn-primary">Search</button>
		</form>

		{#if loading}
			<div class="text-sm text-gcp-muted">Searching…</div>
		{:else if users.length > 0}
			<div class="card overflow-hidden">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-gcp-border text-left text-xs font-medium uppercase text-gcp-muted">
							<th class="px-5 py-3">Username</th>
							<th class="px-5 py-3">Display Name</th>
							<th class="px-5 py-3">Email</th>
							<th class="px-5 py-3">OU</th>
							<th class="px-5 py-3"></th>
						</tr>
					</thead>
					<tbody>
						{#each pagedUsers as u}
							<tr class="border-b border-gcp-border/50 hover:bg-gcp-bg">
								<td class="px-5 py-3 font-mono text-xs font-medium text-gcp-dark">{u.username}</td>
								<td class="px-5 py-3 text-gcp-dark">{u.display_name}</td>
								<td class="px-5 py-3 text-gcp-muted">{u.email || '—'}</td>
								<td class="px-5 py-3">
									<span class="badge bg-gcp-bg text-gcp-muted">{u.ou}</span>
								</td>
								<td class="px-5 py-3">
									<button onclick={() => openQuickAdd(u.username)}
										class="flex items-center gap-1 rounded border border-gcp-border px-2 py-1 text-xs
											text-gcp-dark hover:bg-gcp-blue-light hover:border-gcp-blue hover:text-gcp-blue transition-colors">
										<Plus size={11} />Add to share
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
				{#if users.length > PAGE_SIZE}
					<div class="border-t border-gcp-border px-5 py-2">
						<Pagination
							total={users.length}
							page={userPage}
							pageSize={PAGE_SIZE}
							onPageChange={(p) => (userPage = p)}
						/>
					</div>
				{/if}
			</div>
		{:else if userQuery}
			<div class="text-sm text-gcp-muted">No users found</div>
		{/if}

	{:else}
		<form onsubmit={(e) => { e.preventDefault(); searchGroups(); }} class="mb-4 flex gap-2">
			<input bind:value={groupQuery} placeholder="Search groups…" class="input-field flex-1" />
			<button type="submit" class="btn-primary">Search</button>
		</form>

		{#if loading}
			<div class="text-sm text-gcp-muted">Searching…</div>
		{:else if groups.length > 0}
			<div class="card overflow-hidden">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-gcp-border text-left text-xs font-medium uppercase text-gcp-muted">
							<th class="px-5 py-3">Group</th>
							<th class="px-5 py-3">SMB Name</th>
							<th class="px-5 py-3">Description</th>
							<th class="px-5 py-3"></th>
						</tr>
					</thead>
					<tbody>
						{#each pagedGroups as g}
							<tr class="border-b border-gcp-border/50 hover:bg-gcp-bg">
								<td class="px-5 py-3 font-medium text-gcp-dark">{g.name}</td>
								<td class="px-5 py-3 font-mono text-xs text-gcp-blue">{g.smb_name}</td>
								<td class="px-5 py-3 text-gcp-muted">{g.description || '—'}</td>
								<td class="px-5 py-3">
									<button onclick={() => openQuickAdd(g.smb_name)}
										class="flex items-center gap-1 rounded border border-gcp-border px-2 py-1 text-xs
											text-gcp-dark hover:bg-gcp-blue-light hover:border-gcp-blue hover:text-gcp-blue transition-colors">
										<Plus size={11} />Add to share
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
				{#if groups.length > PAGE_SIZE}
					<div class="border-t border-gcp-border px-5 py-2">
						<Pagination
							total={groups.length}
							page={groupPage}
							pageSize={PAGE_SIZE}
							onPageChange={(p) => (groupPage = p)}
						/>
					</div>
				{/if}
			</div>
		{:else if groupQuery}
			<div class="text-sm text-gcp-muted">No groups found</div>
		{/if}
	{/if}
</div>
