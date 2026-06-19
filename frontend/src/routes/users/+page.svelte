<script lang="ts">
	import { onMount } from 'svelte';
	import { usersApi, type User } from '$lib/api/users';
	import { sharesApi, type Share } from '$lib/api/shares';
	import { toast, toastError } from '$lib/stores/toast.svelte';
	import ConfirmModal from '$lib/components/ConfirmModal.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import { UserRound, Search, X, Plus, Folder } from 'lucide-svelte';

	let users   = $state<User[]>([]);
	let shares  = $state<Share[]>([]);
	let loading = $state(true);
	let showCreate = $state(false);
	let search  = $state('');

	let newUsername = $state('');
	let newPassword = $state('');
	let newFullname = $state('');

	let passwordTarget = $state('');
	let newPw = $state('');

	// Pagination
	let page     = $state(1);
	let pageSize = $state(15);

	// User→shares panel
	let viewingUser = $state<string | null>(null);

	// Confirm delete
	let confirmOpen = $state(false);
	let confirmUser = $state('');

	$effect(() => { search; page = 1; });

	const filtered = $derived.by(() => {
		if (!search.trim()) return users;
		const q = search.toLowerCase();
		return users.filter(u =>
			u.username.toLowerCase().includes(q) ||
			(u.fullname ?? '').toLowerCase().includes(q)
		);
	});

	const paged = $derived(filtered.slice((page - 1) * pageSize, page * pageSize));

	const userShares = $derived<{ share: Share; perms: string[] }[]>(
		viewingUser
			? shares.reduce<{ share: Share; perms: string[] }[]>((acc, s) => {
				const u = viewingUser!;
				const perms: string[] = [];
				if ((s.admin_users   ?? []).includes(u)) perms.push('Admin');
				if ((s.write_list    ?? []).includes(u)) perms.push('Write');
				if ((s.read_list     ?? []).includes(u)) perms.push('Read');
				if ((s.valid_users   ?? []).includes(u)) perms.push('Valid');
				if ((s.invalid_users ?? []).includes(u)) perms.push('Blocked');
				if (perms.length) acc.push({ share: s, perms });
				return acc;
			}, [])
			: []
	);

	async function load() {
		loading = true;
		try {
			const [ur, sr] = await Promise.all([usersApi.list(), sharesApi.list()]);
			users  = ur.users ?? [];
			shares = sr.shares ?? [];
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to load');
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

	function askDelete(username: string) {
		confirmUser = username;
		confirmOpen = true;
	}

	async function deleteUser() {
		confirmOpen = false;
		try {
			await usersApi.delete(confirmUser);
			toast(`User '${confirmUser}' deleted`);
			if (viewingUser === confirmUser) viewingUser = null;
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

<ConfirmModal
	open={confirmOpen}
	title="Delete user"
	message="Delete local Samba user '{confirmUser}'?"
	confirmLabel="Delete"
	danger={true}
	onconfirm={deleteUser}
	oncancel={() => (confirmOpen = false)}
/>

<div>
	<!-- Header + create -->
	<div class="mb-4 flex items-center gap-3">
		<h1 class="page-title flex-1">Local Samba Users</h1>
		<button onclick={() => (showCreate = !showCreate)}
			class="flex items-center gap-1 rounded border border-gcp-border bg-white px-3 py-1.5
				text-xs text-gcp-dark hover:bg-gcp-bg transition-colors">
			<Plus size={12} />New User
		</button>
	</div>

	{#if showCreate}
		<div class="card mb-4 p-4">
			<h2 class="mb-3 text-xs font-semibold uppercase tracking-wide text-gcp-muted">Create User</h2>
			<form onsubmit={(e) => { e.preventDefault(); createUser(); }} class="flex flex-wrap gap-2">
				<input bind:value={newUsername} placeholder="Username" required class="input-field w-36" />
				<input type="password" bind:value={newPassword} placeholder="Password" required class="input-field w-40" />
				<input bind:value={newFullname} placeholder="Full name (optional)" class="input-field w-48" />
				<button type="submit" class="btn-primary text-xs py-1.5">Create</button>
				<button type="button" onclick={() => (showCreate = false)} class="btn-secondary text-xs py-1.5">Cancel</button>
			</form>
		</div>
	{/if}

	<!-- Search -->
	<div class="relative mb-3 max-w-sm">
		<Search size={13} class="absolute left-2.5 top-1/2 -translate-y-1/2 text-gcp-muted" />
		<input bind:value={search} placeholder="Search users…" class="input-field w-full pl-8 text-xs" />
		{#if search}
			<button onclick={() => (search = '')} class="absolute right-2 top-1/2 -translate-y-1/2 text-gcp-muted hover:text-gcp-dark">
				<X size={12} />
			</button>
		{/if}
	</div>

	<div class="flex gap-5">
		<!-- User table -->
		<div class="min-w-0 flex-1 card overflow-hidden">
			{#if loading}
				<div class="p-5 text-sm text-gcp-muted">Loading…</div>
			{:else if filtered.length === 0}
				<div class="p-5 text-sm text-gcp-muted">{search ? 'No matches' : 'No users found'}</div>
			{:else}
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-gcp-border text-left text-xs font-medium uppercase text-gcp-muted">
							<th class="px-5 py-3">Username</th>
							<th class="px-5 py-3">UID</th>
							<th class="px-5 py-3">Full Name</th>
							<th class="px-5 py-3">Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each paged as u}
							<tr class="border-b border-gcp-border/50 hover:bg-gcp-bg">
								<td class="px-5 py-3">
									<button
										onclick={() => viewingUser = viewingUser === u.username ? null : u.username}
										class="flex items-center gap-2 font-medium
											{viewingUser === u.username ? 'text-gcp-blue' : 'text-gcp-dark'}"
									>
										<UserRound size={14} class="flex-none text-gcp-muted" />{u.username}
									</button>
								</td>
								<td class="px-5 py-3 text-gcp-muted font-mono text-xs">{u.uid}</td>
								<td class="px-5 py-3 text-gcp-muted">{u.fullname || '—'}</td>
								<td class="px-5 py-3">
									<div class="flex items-center gap-2">
										{#if passwordTarget === u.username}
											<input type="password" bind:value={newPw} placeholder="New password"
												class="input-field w-32 text-xs" />
											<button onclick={() => changePassword(u.username)}
												class="rounded bg-gcp-green px-2 py-1 text-xs text-white hover:opacity-90">
												Save
											</button>
											<button onclick={() => { passwordTarget = ''; newPw = ''; }}
												class="btn-secondary text-xs px-2 py-1">Cancel</button>
										{:else}
											<button onclick={() => { passwordTarget = u.username; newPw = ''; }}
												class="rounded border border-gcp-border px-2 py-1 text-xs text-gcp-dark
													hover:bg-gcp-bg transition-colors">
												Password
											</button>
											<button onclick={() => askDelete(u.username)}
												class="rounded px-2 py-1 text-xs text-gcp-red hover:bg-red-50 transition-colors">
												Delete
											</button>
										{/if}
									</div>
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
							pageSizeOptions={[10, 15, 25, 50]}
							onPageChange={(p) => (page = p)}
							onPageSizeChange={(s) => { pageSize = s; page = 1; }}
						/>
					</div>
				{/if}
			{/if}
		</div>

		<!-- User→shares panel -->
		{#if viewingUser}
			<div class="w-64 flex-none card p-4">
				<div class="mb-3 flex items-center justify-between">
					<h3 class="text-xs font-semibold uppercase tracking-wide text-gcp-muted">
						Shares for {viewingUser}
					</h3>
					<button onclick={() => (viewingUser = null)} class="text-gcp-muted hover:text-gcp-dark">
						<X size={14} />
					</button>
				</div>
				{#if userShares.length === 0}
					<p class="text-xs text-gcp-muted italic">Not in any share</p>
				{:else}
					<div class="space-y-2">
						{#each userShares as { share, perms }}
							<div class="rounded border border-gcp-border p-2.5">
								<div class="flex items-center gap-1.5 text-xs font-medium text-gcp-dark mb-1.5">
									<Folder size={12} class="flex-none text-gcp-blue" />
									<a href="/shares" class="hover:underline">{share.name}</a>
								</div>
								<div class="flex flex-wrap gap-1">
									{#each perms as p}
										<span class="badge
											{p === 'Admin' ? 'bg-purple-100 text-purple-800' :
											 p === 'Write' ? 'bg-green-100 text-gcp-green' :
											 p === 'Read' ? 'bg-gcp-blue-light text-gcp-blue' :
											 p === 'Blocked' ? 'bg-red-100 text-gcp-red' :
											 'bg-gray-100 text-gcp-muted'}">
											{p}
										</span>
									{/each}
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		{/if}
	</div>
</div>
