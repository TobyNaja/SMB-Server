<script lang="ts">
	import { onMount } from 'svelte';
	import { builtinApi, type BuiltinGroup } from '$lib/api/builtin';
	import { toast, toastError } from '$lib/stores/toast.svelte';
	import ConfirmModal from '$lib/components/ConfirmModal.svelte';
	import { ShieldCheck, ChevronDown, ChevronRight, X, Plus } from 'lucide-svelte';

	let groups   = $state<BuiltinGroup[]>([]);
	let loading  = $state(true);
	let expanded = $state<string | null>(null);
	let newMember = $state('');
	let addingTo  = $state('');

	// Remove confirm
	let confirmOpen   = $state(false);
	let confirmGroup  = $state('');
	let confirmMember = $state('');

	async function load() {
		loading = true;
		try {
			const r = await builtinApi.list();
			groups = r.groups ?? [];
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to load builtin groups');
		} finally {
			loading = false;
		}
	}

	function askRemove(groupName: string, member: string) {
		confirmGroup  = groupName;
		confirmMember = member;
		confirmOpen   = true;
	}

	async function removeMember() {
		confirmOpen = false;
		try {
			await builtinApi.removeMember(confirmGroup, confirmMember);
			toast(`Removed '${confirmMember}' from ${confirmGroup}`);
			await load();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to remove member');
		}
	}

	async function addMember(groupName: string) {
		if (!newMember.trim()) return;
		try {
			await builtinApi.addMember(groupName, newMember.trim());
			toast(`Added '${newMember}' to ${groupName}`);
			newMember = '';
			addingTo  = '';
			await load();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to add member');
		}
	}

	onMount(load);
</script>

<ConfirmModal
	open={confirmOpen}
	title="Remove member"
	message="Remove '{confirmMember}' from BUILTIN\{confirmGroup}?"
	confirmLabel="Remove"
	danger={true}
	onconfirm={removeMember}
	oncancel={() => (confirmOpen = false)}
/>

<div>
	<h1 class="page-title mb-1">BUILTIN Groups</h1>
	<p class="mb-5 text-xs text-gcp-muted">
		Windows BUILTIN\ group membership — synced via <code>net sam addmem/delmem</code>.
	</p>

	{#if loading}
		<div class="text-sm text-gcp-muted">Loading…</div>
	{:else}
		<div class="space-y-2">
			{#each groups as group}
				<div class="card overflow-hidden">
					<button
						onclick={() => (expanded = expanded === group.name ? null : group.name)}
						class="flex w-full items-center justify-between px-5 py-4 text-left hover:bg-gcp-bg transition-colors"
					>
						<div class="flex items-center gap-3">
							<ShieldCheck size={16} class="flex-none text-gcp-blue" />
							<div>
								<div class="text-sm font-medium text-gcp-dark">BUILTIN\{group.name}</div>
								<div class="text-xs text-gcp-muted">{group.description}</div>
							</div>
						</div>
						<div class="flex items-center gap-2">
							<span class="badge bg-gcp-bg text-gcp-muted">
								{group.members.length} member{group.members.length !== 1 ? 's' : ''}
							</span>
							{#if expanded === group.name}
								<ChevronDown size={14} class="text-gcp-muted" />
							{:else}
								<ChevronRight size={14} class="text-gcp-muted" />
							{/if}
						</div>
					</button>

					{#if expanded === group.name}
						<div class="border-t border-gcp-border bg-gcp-bg px-5 pb-4 pt-3">
							{#if group.members.length === 0}
								<p class="mb-3 text-xs text-gcp-muted italic">No members</p>
							{:else}
								<div class="mb-3 flex flex-wrap gap-1.5">
									{#each group.members as m}
										<span class="flex items-center gap-1 rounded border border-gcp-border bg-white
											px-2.5 py-1 text-xs text-gcp-dark">
											{m}
											<button onclick={() => askRemove(group.name, m)}
												class="ml-1 text-gcp-muted hover:text-gcp-red transition-colors">
												<X size={10} />
											</button>
										</span>
									{/each}
								</div>
							{/if}

							{#if addingTo === group.name}
								<form onsubmit={(e) => { e.preventDefault(); addMember(group.name); }}
									class="flex gap-2">
									<input bind:value={newMember} placeholder="username or IT\user"
										class="input-field flex-1 text-xs" />
									<button type="submit"
										class="rounded bg-gcp-green px-3 py-1.5 text-xs text-white hover:opacity-90">
										Add
									</button>
									<button type="button" onclick={() => { addingTo = ''; newMember = ''; }}
										class="btn-secondary text-xs px-3 py-1.5">Cancel</button>
								</form>
							{:else}
								<button onclick={() => { addingTo = group.name; newMember = ''; }}
									class="flex items-center gap-1.5 rounded border border-gcp-border bg-white
										px-3 py-1.5 text-xs text-gcp-dark hover:bg-gcp-bg transition-colors">
									<Plus size={11} />Add member
								</button>
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>
