<script lang="ts">
	import { onMount } from 'svelte';
	import { builtinApi, type BuiltinGroup } from '$lib/api/builtin';
	import { toast, toastError } from '$lib/stores/toast.svelte';

	let groups = $state<BuiltinGroup[]>([]);
	let loading = $state(true);
	let expanded = $state<string | null>(null);
	let newMember = $state('');
	let addingTo = $state('');

	async function load() {
		loading = true;
		try {
			const r = await builtinApi.list();
			groups = r.groups;
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to load builtin groups');
		} finally {
			loading = false;
		}
	}

	async function removeMember(groupName: string, member: string) {
		try {
			await builtinApi.removeMember(groupName, member);
			toast(`Removed '${member}' from ${groupName}`);
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
			addingTo = '';
			await load();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to add member');
		}
	}

	onMount(load);
</script>

<div>
	<h1 class="mb-2 text-lg font-semibold text-gray-800 dark:text-white">BUILTIN Groups</h1>
	<p class="mb-6 text-sm text-gray-500 dark:text-gray-400">
		Windows BUILTIN\ group membership. Changes are synced to Samba via <code class="text-xs">net sam addmem/delmem</code>.
	</p>

	{#if loading}
		<div class="text-sm text-gray-400">Loading…</div>
	{:else}
		<div class="space-y-3">
			{#each groups as group}
				<div class="rounded-xl bg-white shadow-sm dark:bg-gray-800">
					<button
						onclick={() => (expanded = expanded === group.name ? null : group.name)}
						class="flex w-full items-center justify-between px-5 py-4 text-left"
					>
						<div class="flex items-center gap-3">
							<span class="text-base">🔐</span>
							<div>
								<div class="font-medium text-gray-800 dark:text-white">BUILTIN\{group.name}</div>
								<div class="text-xs text-gray-400">{group.description}</div>
							</div>
						</div>
						<div class="flex items-center gap-2">
							<span class="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-400">
								{group.members.length} member{group.members.length !== 1 ? 's' : ''}
							</span>
							<span class="text-gray-400">{expanded === group.name ? '▾' : '▸'}</span>
						</div>
					</button>

					{#if expanded === group.name}
						<div class="border-t border-gray-100 px-5 pb-4 pt-3 dark:border-gray-700">
							{#if group.members.length === 0}
								<p class="mb-3 text-xs text-gray-400 italic">No members</p>
							{:else}
								<div class="mb-3 flex flex-wrap gap-2">
									{#each group.members as m}
										<span class="flex items-center gap-1 rounded-full bg-blue-50 px-3 py-1 text-xs text-blue-800 dark:bg-blue-900/30 dark:text-blue-300">
											{m}
											<button onclick={() => removeMember(group.name, m)}
												class="ml-1 text-blue-400 hover:text-red-500">✕</button>
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
										class="rounded bg-green-600 px-3 py-1.5 text-xs text-white hover:bg-green-700">
										Add
									</button>
									<button type="button" onclick={() => { addingTo = ''; newMember = ''; }}
										class="rounded bg-gray-200 px-3 py-1.5 text-xs text-gray-700 hover:bg-gray-300 dark:bg-gray-700 dark:text-gray-300">
										Cancel
									</button>
								</form>
							{:else}
								<button onclick={() => { addingTo = group.name; newMember = ''; }}
									class="rounded bg-gray-100 px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300">
									+ Add member
								</button>
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

