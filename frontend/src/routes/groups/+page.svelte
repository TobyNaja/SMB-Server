<script lang="ts">
	import { onMount } from 'svelte';
	import { groupsApi } from '$lib/api/groups';
	import { toast, toastError } from '$lib/stores/toast.svelte';

	let groups = $state<string[]>([]);
	let loading = $state(true);
	let newGroup = $state('');
	let addMemberGroup = $state('');
	let addMemberUser = $state('');

	async function load() {
		loading = true;
		try {
			const r = await groupsApi.list();
			groups = r.groups;
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
	<h1 class="mb-6 text-lg font-semibold text-gray-800 dark:text-white">Local Linux Groups</h1>

	<div class="mb-6 flex gap-3">
		<form onsubmit={(e) => { e.preventDefault(); createGroup(); }} class="flex gap-2">
			<input bind:value={newGroup} placeholder="New group name" required
				class="input-field w-52" />
			<button type="submit" class="rounded bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700">
				Create Group
			</button>
		</form>
	</div>

	<div class="mb-6 rounded-xl bg-white p-5 shadow-sm dark:bg-gray-800">
		<h2 class="mb-3 text-sm font-medium text-gray-700 dark:text-gray-300">Add User to Group</h2>
		<form onsubmit={(e) => { e.preventDefault(); addMember(); }} class="flex flex-wrap gap-3">
			<select bind:value={addMemberGroup} class="select-field w-48">
				<option value="">Select group…</option>
				{#each groups as g}<option value={g}>{g}</option>{/each}
			</select>
			<input bind:value={addMemberUser} placeholder="Username" required class="input-field w-44" />
			<button type="submit" class="rounded bg-green-600 px-4 py-2 text-sm text-white hover:bg-green-700">
				Add Member
			</button>
		</form>
	</div>

	<div class="rounded-xl bg-white shadow-sm dark:bg-gray-800">
		{#if loading}
			<div class="p-6 text-sm text-gray-400">Loading…</div>
		{:else if groups.length === 0}
			<div class="p-6 text-sm text-gray-400">No groups found</div>
		{:else}
			<div class="divide-y divide-gray-50 dark:divide-gray-700">
				{#each groups as g}
					<div class="flex items-center justify-between px-6 py-3">
						<span class="text-sm font-medium text-gray-800 dark:text-white">👥 {g}</span>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>

