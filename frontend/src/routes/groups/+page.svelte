<script lang="ts">
	import { onMount } from 'svelte';
	import { groupsApi } from '$lib/api/groups';
	import { toast, toastError } from '$lib/stores/toast.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import { Users, Search, X, Plus } from 'lucide-svelte';

	let groups = $state<string[]>([]);
	let loading = $state(true);
	let newGroup = $state('');
	let search = $state('');
	let addMemberGroup = $state('');
	let addMemberUser  = $state('');

	let page     = $state(1);
	let pageSize = $state(20);

	$effect(() => { search; page = 1; });

	const filtered = $derived.by(() => {
		if (!search.trim()) return groups;
		return groups.filter(g => g.toLowerCase().includes(search.toLowerCase()));
	});

	const paged = $derived(filtered.slice((page - 1) * pageSize, page * pageSize));

	async function load() {
		loading = true;
		try {
			const r = await groupsApi.list();
			groups = r.groups ?? [];
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to load groups');
		} finally {
			loading = false;
		}
	}

	async function createGroup() {
		if (!newGroup) return;
		try {
			await groupsApi.create(newGroup);
			toast(`Group '${newGroup}' created`);
			newGroup = '';
			await load();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to create group');
		}
	}

	async function addMember() {
		if (!addMemberGroup || !addMemberUser) return;
		try {
			await groupsApi.addMember(addMemberGroup, addMemberUser);
			toast(`Added '${addMemberUser}' to '${addMemberGroup}'`);
			addMemberUser = '';
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to add member');
		}
	}

	onMount(load);
</script>

<div>
	<h1 class="page-title mb-4">Local Linux Groups</h1>

	<!-- Create + add member forms -->
	<div class="card mb-4 p-4">
		<div class="flex flex-wrap gap-6">
			<div>
				<h2 class="mb-2 text-xs font-semibold uppercase tracking-wide text-gcp-muted">New Group</h2>
				<form onsubmit={(e) => { e.preventDefault(); createGroup(); }} class="flex gap-2">
					<input bind:value={newGroup} placeholder="Group name" required class="input-field w-44" />
					<button type="submit" class="btn-primary text-xs py-1.5">
						<Plus size={12} class="inline mr-1" />Create
					</button>
				</form>
			</div>
			<div>
				<h2 class="mb-2 text-xs font-semibold uppercase tracking-wide text-gcp-muted">Add User to Group</h2>
				<form onsubmit={(e) => { e.preventDefault(); addMember(); }} class="flex flex-wrap gap-2">
					<select bind:value={addMemberGroup} class="select-field w-40">
						<option value="">Select group…</option>
						{#each groups as g}<option value={g}>{g}</option>{/each}
					</select>
					<input bind:value={addMemberUser} placeholder="Username" required class="input-field w-36" />
					<button type="submit" class="rounded bg-gcp-green px-3 py-1.5 text-xs text-white hover:opacity-90 disabled:opacity-60 transition-colors">
						Add Member
					</button>
				</form>
			</div>
		</div>
	</div>

	<!-- Search -->
	<div class="relative mb-3 max-w-sm">
		<Search size={13} class="absolute left-2.5 top-1/2 -translate-y-1/2 text-gcp-muted" />
		<input bind:value={search} placeholder="Search groups…" class="input-field w-full pl-8 text-xs" />
		{#if search}
			<button onclick={() => (search = '')} class="absolute right-2 top-1/2 -translate-y-1/2 text-gcp-muted hover:text-gcp-dark">
				<X size={12} />
			</button>
		{/if}
	</div>

	<div class="card overflow-hidden">
		{#if loading}
			<div class="p-5 text-sm text-gcp-muted">Loading…</div>
		{:else if filtered.length === 0}
			<div class="p-5 text-sm text-gcp-muted">{search ? 'No matches' : 'No groups found'}</div>
		{:else}
			<table class="w-full text-sm">
				<thead>
					<tr class="border-b border-gcp-border text-left text-xs font-medium uppercase text-gcp-muted">
						<th class="px-5 py-3">Group Name</th>
					</tr>
				</thead>
				<tbody>
					{#each paged as g}
						<tr class="border-b border-gcp-border/50 hover:bg-gcp-bg">
							<td class="px-5 py-3">
								<span class="flex items-center gap-2 font-medium text-gcp-dark">
									<Users size={14} class="flex-none text-gcp-muted" />{g}
								</span>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
			{#if filtered.length > pageSize}
				<div class="border-t border-gcp-border px-5 pb-3">
					<Pagination
						total={filtered.length}
						{page}
						{pageSize}
						pageSizeOptions={[10, 20, 50]}
						onPageChange={(p) => (page = p)}
						onPageSizeChange={(s) => { pageSize = s; page = 1; }}
					/>
				</div>
			{/if}
		{/if}
	</div>
</div>
