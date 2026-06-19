<script lang="ts">
	import { onMount } from 'svelte';
	import { usersApi, type User } from '$lib/api/users';
	import { toast, toastError } from '$lib/stores/toast.svelte';

	let users = $state<User[]>([]);
	let loading = $state(true);
	let showCreate = $state(false);

	let newUsername = $state('');
	let newPassword = $state('');
	let newFullname = $state('');

	let passwordTarget = $state('');
	let newPw = $state('');

	async function load() {
		loading = true;
		try {
			const r = await usersApi.list();
			users = r.users;
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to load users');
		} finally {
			loading = false;
		}
	}

	async function createUser() {
		try {
			await usersApi.create(newUsername, newPassword, newFullname);
			toast(`User '${newUsername}' created`);
			newUsername = ''; newPassword = ''; newFullname = ''; showCreate = false;
			await load();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to create user');
		}
	}

	async function deleteUser(username: string) {
		if (!confirm(`Delete user '${username}'?`)) return;
		try {
			await usersApi.delete(username);
			toast(`User '${username}' deleted`);
			await load();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to delete user');
		}
	}

	async function changePassword(username: string) {
		if (!newPw) return;
		try {
			await usersApi.setPassword(username, newPw);
			toast('Password updated');
			passwordTarget = ''; newPw = '';
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to set password');
		}
	}

	onMount(load);
</script>

<div>
	<div class="mb-6 flex items-center justify-between">
		<h1 class="text-lg font-semibold text-gray-800 dark:text-white">Local Samba Users</h1>
		<button onclick={() => (showCreate = !showCreate)}
			class="rounded bg-blue-600 px-3 py-1.5 text-sm text-white hover:bg-blue-700">
			+ New User
		</button>
	</div>

	{#if showCreate}
		<div class="mb-6 rounded-xl bg-white p-5 shadow-sm dark:bg-gray-800">
			<h2 class="mb-4 text-sm font-medium text-gray-700 dark:text-gray-300">Create User</h2>
			<form onsubmit={(e) => { e.preventDefault(); createUser(); }} class="flex flex-wrap gap-3">
				<input bind:value={newUsername} placeholder="Username" required class="input-field w-40" />
				<input type="password" bind:value={newPassword} placeholder="Password" required class="input-field w-44" />
				<input bind:value={newFullname} placeholder="Full name (optional)" class="input-field w-52" />
				<button type="submit" class="rounded bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700">
					Create
				</button>
				<button type="button" onclick={() => (showCreate = false)}
					class="rounded bg-gray-200 px-4 py-2 text-sm text-gray-700 hover:bg-gray-300 dark:bg-gray-700 dark:text-gray-300">
					Cancel
				</button>
			</form>
		</div>
	{/if}

	<div class="rounded-xl bg-white shadow-sm dark:bg-gray-800">
		{#if loading}
			<div class="p-6 text-sm text-gray-400">Loading…</div>
		{:else if users.length === 0}
			<div class="p-6 text-sm text-gray-400">No users found</div>
		{:else}
			<table class="w-full text-sm">
				<thead>
					<tr class="border-b border-gray-100 text-left text-xs font-medium uppercase text-gray-400 dark:border-gray-700">
						<th class="px-6 py-3">Username</th>
						<th class="px-6 py-3">UID</th>
						<th class="px-6 py-3">Full Name</th>
						<th class="px-6 py-3">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each users as u}
						<tr class="border-b border-gray-50 hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-700/50">
							<td class="px-6 py-3 font-medium text-gray-800 dark:text-white">👤 {u.username}</td>
							<td class="px-6 py-3 text-gray-500">{u.uid}</td>
							<td class="px-6 py-3 text-gray-500">{u.fullname || '—'}</td>
							<td class="px-6 py-3">
								<div class="flex items-center gap-2">
									{#if passwordTarget === u.username}
										<input type="password" bind:value={newPw} placeholder="New password"
											class="input-field w-36" />
										<button onclick={() => changePassword(u.username)}
											class="rounded bg-green-600 px-2 py-1 text-xs text-white hover:bg-green-700">
											Save
										</button>
										<button onclick={() => { passwordTarget = ''; newPw = ''; }}
											class="rounded bg-gray-200 px-2 py-1 text-xs text-gray-700 hover:bg-gray-300">
											Cancel
										</button>
									{:else}
										<button onclick={() => { passwordTarget = u.username; newPw = ''; }}
											class="rounded bg-amber-500 px-2 py-1 text-xs text-white hover:bg-amber-600">
											Password
										</button>
										<button onclick={() => deleteUser(u.username)}
											class="rounded bg-red-600 px-2 py-1 text-xs text-white hover:bg-red-700">
											Delete
										</button>
									{/if}
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>
</div>

