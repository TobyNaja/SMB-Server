<script lang="ts">
	import { onMount } from 'svelte';
	import { adApi, type ADUser, type ADGroup, type ADStatus } from '$lib/api/ad';
	import { toastError } from '$lib/stores/toast.svelte';

	let status = $state<ADStatus | null>(null);
	let users = $state<ADUser[]>([]);
	let groups = $state<ADGroup[]>([]);
	let loading = $state(false);
	let tab = $state<'users' | 'groups'>('users');

	let userQuery = $state('');
	let groupQuery = $state('');

	onMount(async () => {
		try {
			status = await adApi.status();
		} catch (e) {
			toastError('Failed to check AD status');
		}
	});

	async function searchUsers() {
		loading = true;
		try {
			const r = await adApi.searchUsers(userQuery);
			users = r.users;
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
			groups = r.groups;
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'AD search failed');
		} finally {
			loading = false;
		}
	}
</script>

<div>
	<h1 class="mb-4 text-lg font-semibold text-gray-800 dark:text-white">Active Directory</h1>

	<!-- Status badge -->
	{#if status}
		<div class="mb-6 flex items-center gap-3 rounded-xl bg-white p-4 shadow-sm dark:bg-gray-800">
			<div class="h-3 w-3 rounded-full {status.connected ? 'bg-green-500' : 'bg-red-500'}"></div>
			<div class="text-sm">
				<span class="font-medium text-gray-700 dark:text-gray-300">{status.domain}</span>
				<span class="ml-2 text-gray-400">{status.ldap_server}</span>
			</div>
			{#if !status.connected && status.error}
				<span class="ml-auto text-xs text-red-500">{status.error}</span>
			{/if}
		</div>
	{/if}

	<!-- Tabs -->
	<div class="mb-4 flex gap-2 border-b border-gray-200 dark:border-gray-700">
		<button onclick={() => (tab = 'users')}
			class="px-4 py-2 text-sm font-medium transition-colors
				{tab === 'users' ? 'border-b-2 border-blue-600 text-blue-600' : 'text-gray-500 hover:text-gray-700'}">
			Users
		</button>
		<button onclick={() => (tab = 'groups')}
			class="px-4 py-2 text-sm font-medium transition-colors
				{tab === 'groups' ? 'border-b-2 border-blue-600 text-blue-600' : 'text-gray-500 hover:text-gray-700'}">
			Groups
		</button>
	</div>

	{#if tab === 'users'}
		<form onsubmit={(e) => { e.preventDefault(); searchUsers(); }} class="mb-4 flex gap-2">
			<input bind:value={userQuery} placeholder="Search by username, name, or email…"
				class="input-field flex-1" />
			<button type="submit" class="rounded bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700">
				Search
			</button>
		</form>

		{#if loading}
			<div class="text-sm text-gray-400">Searching…</div>
		{:else if users.length > 0}
			<div class="rounded-xl bg-white shadow-sm dark:bg-gray-800">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-gray-100 text-left text-xs font-medium uppercase text-gray-400 dark:border-gray-700">
							<th class="px-6 py-3">Username</th>
							<th class="px-6 py-3">Display Name</th>
							<th class="px-6 py-3">Email</th>
							<th class="px-6 py-3">OU</th>
						</tr>
					</thead>
					<tbody>
						{#each users as u}
							<tr class="border-b border-gray-50 hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-700/50">
								<td class="px-6 py-3 font-mono text-xs font-medium text-gray-800 dark:text-white">{u.username}</td>
								<td class="px-6 py-3 text-gray-700 dark:text-gray-300">{u.display_name}</td>
								<td class="px-6 py-3 text-gray-500">{u.email || '—'}</td>
								<td class="px-6 py-3">
									<span class="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-400">{u.ou}</span>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{:else if userQuery}
			<div class="text-sm text-gray-400">No users found</div>
		{/if}
	{:else}
		<form onsubmit={(e) => { e.preventDefault(); searchGroups(); }} class="mb-4 flex gap-2">
			<input bind:value={groupQuery} placeholder="Search groups…"
				class="input-field flex-1" />
			<button type="submit" class="rounded bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700">
				Search
			</button>
		</form>

		{#if loading}
			<div class="text-sm text-gray-400">Searching…</div>
		{:else if groups.length > 0}
			<div class="rounded-xl bg-white shadow-sm dark:bg-gray-800">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-gray-100 text-left text-xs font-medium uppercase text-gray-400 dark:border-gray-700">
							<th class="px-6 py-3">Group</th>
							<th class="px-6 py-3">SMB Name</th>
							<th class="px-6 py-3">Description</th>
						</tr>
					</thead>
					<tbody>
						{#each groups as g}
							<tr class="border-b border-gray-50 hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-700/50">
								<td class="px-6 py-3 font-medium text-gray-800 dark:text-white">{g.name}</td>
								<td class="px-6 py-3 font-mono text-xs text-blue-600 dark:text-blue-400">{g.smb_name}</td>
								<td class="px-6 py-3 text-gray-500">{g.description || '—'}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{:else if groupQuery}
			<div class="text-sm text-gray-400">No groups found</div>
		{/if}
	{/if}
</div>

