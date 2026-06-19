<script lang="ts">
	import { onMount } from 'svelte';
	import { sharesApi, type Share, type PermissionType } from '$lib/api/shares';
	import { toast, toastError } from '$lib/stores/toast.svelte';

	let shares = $state<Share[]>([]);
	let loading = $state(true);
	let selected = $state<Share | null>(null);
	let showCreate = $state(false);

	// Create form
	let newName = $state('');
	let newPath = $state('');
	let newComment = $state('');
	let newBrowseable = $state(true);
	let newGuestOk = $state(false);

	// Permission editor
	let permInput = $state('');
	let permType = $state<PermissionType>('valid_users');

	const permLabels: Record<PermissionType, { label: string; color: string }> = {
		valid_users:   { label: 'Valid Users',   color: 'bg-gray-600' },
		write_list:    { label: 'Write List',    color: 'bg-green-700' },
		read_list:     { label: 'Read List',     color: 'bg-blue-700' },
		admin_users:   { label: 'Admin Users',   color: 'bg-purple-700' },
		invalid_users: { label: 'Blocked',       color: 'bg-red-700' }
	};

	async function load() {
		loading = true;
		try {
			const r = await sharesApi.list();
			shares = r.shares;
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to load shares');
		} finally {
			loading = false;
		}
	}

	async function createShare() {
		if (!newName || !newPath) return;
		try {
			await sharesApi.create({ name: newName, path: newPath, comment: newComment, browseable: newBrowseable, guest_ok: newGuestOk });
			toast(`Share '${newName}' created`);
			newName = ''; newPath = ''; newComment = ''; showCreate = false;
			await load();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to create share');
		}
	}

	async function deleteShare(name: string) {
		if (!confirm(`Delete share '${name}'?`)) return;
		try {
			await sharesApi.delete(name);
			toast(`Share '${name}' deleted`);
			if (selected?.name === name) selected = null;
			await load();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to delete share');
		}
	}

	async function selectShare(name: string) {
		try {
			selected = await sharesApi.get(name);
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to load share');
		}
	}

	async function savePermissions() {
		if (!selected) return;
		const users = permInput.split(/[\s,]+/).map(u => u.trim()).filter(Boolean);
		try {
			await sharesApi.setPermissions(selected.name, permType, users);
			toast('Permissions updated');
			await selectShare(selected.name);
			await load();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to update permissions');
		}
	}

	function getUserList(share: Share, type: PermissionType): string[] {
		const map: Record<PermissionType, string[]> = {
			valid_users:   share.valid_users,
			write_list:    share.write_list,
			read_list:     share.read_list,
			admin_users:   share.admin_users,
			invalid_users: share.invalid_users
		};
		return map[type] ?? [];
	}

	onMount(load);
</script>

<div class="flex h-full gap-6">
	<!-- Share list -->
	<div class="w-72 flex-none">
		<div class="mb-4 flex items-center justify-between">
			<h1 class="text-lg font-semibold text-gray-800 dark:text-white">Shares</h1>
			<button
				onclick={() => (showCreate = !showCreate)}
				class="rounded bg-blue-600 px-3 py-1 text-xs text-white hover:bg-blue-700"
			>+ New</button>
		</div>

		{#if showCreate}
			<div class="mb-4 rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
				<h2 class="mb-3 text-sm font-medium text-gray-700 dark:text-gray-300">New Share</h2>
				<form onsubmit={(e) => { e.preventDefault(); createShare(); }} class="space-y-2">
					<input bind:value={newName} placeholder="Share name" required
						class="input-field w-full" />
					<input bind:value={newPath} placeholder="/srv/shares/name" required
						class="input-field w-full" />
					<input bind:value={newComment} placeholder="Description (optional)"
						class="input-field w-full" />
					<label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
						<input type="checkbox" bind:checked={newBrowseable} />
						Browseable
					</label>
					<label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
						<input type="checkbox" bind:checked={newGuestOk} />
						Guest OK
					</label>
					<div class="flex gap-2">
						<button type="submit" class="flex-1 rounded bg-blue-600 py-1.5 text-xs text-white hover:bg-blue-700">
							Create
						</button>
						<button type="button" onclick={() => (showCreate = false)}
							class="flex-1 rounded bg-gray-200 py-1.5 text-xs text-gray-700 hover:bg-gray-300 dark:bg-gray-700 dark:text-gray-300">
							Cancel
						</button>
					</div>
				</form>
			</div>
		{/if}

		{#if loading}
			<div class="text-sm text-gray-400">Loading…</div>
		{:else if shares.length === 0}
			<div class="text-sm text-gray-400">No shares found</div>
		{:else}
			<div class="space-y-1">
				{#each shares as share}
					<!-- svelte-ignore a11y_interactive_supports_focus -->
					<div
						role="button"
						tabindex="0"
						onclick={() => selectShare(share.name)}
						onkeypress={(e) => e.key === 'Enter' && selectShare(share.name)}
						class="w-full cursor-pointer rounded-lg px-3 py-2.5 text-left text-sm transition-colors
							{selected?.name === share.name
							? 'bg-blue-600 text-white'
							: 'bg-white text-gray-700 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700'}"
					>
						<div class="flex items-center justify-between">
							<span class="font-medium">📁 {share.name}</span>
							<button
								onclick={(e) => { e.stopPropagation(); deleteShare(share.name); }}
								class="rounded px-1.5 py-0.5 text-xs opacity-60 hover:bg-red-600 hover:text-white hover:opacity-100"
							>✕</button>
						</div>
						<div class="mt-0.5 truncate text-xs opacity-70">{share.path}</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Share detail / permission editor -->
	<div class="flex-1">
		{#if selected}
			<div class="rounded-xl bg-white shadow-sm dark:bg-gray-800">
				<div class="border-b border-gray-100 px-6 py-4 dark:border-gray-700">
					<h2 class="font-semibold text-gray-800 dark:text-white">📁 {selected.name}</h2>
					<p class="text-sm text-gray-500 dark:text-gray-400">{selected.path}</p>
				</div>

				<div class="p-6">
					<!-- Info chips -->
					<div class="mb-6 flex flex-wrap gap-2 text-xs">
						{#if selected.read_only}
							<span class="badge bg-amber-100 text-amber-800">Read-only</span>
						{:else}
							<span class="badge bg-green-100 text-green-800">Writable</span>
						{/if}
						{#if selected.browseable}
							<span class="badge bg-blue-100 text-blue-800">Browseable</span>
						{/if}
						{#if selected.guest_ok}
							<span class="badge bg-orange-100 text-orange-800">Guest OK</span>
						{/if}
						{#if selected.abse}
							<span class="badge bg-purple-100 text-purple-800">ABSE</span>
						{/if}
					</div>

					<!-- Current user lists -->
					<div class="mb-6 grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
						{#each Object.entries(permLabels) as [type, meta]}
							{@const users = getUserList(selected, type as PermissionType)}
							<div class="rounded-lg border border-gray-100 p-3 dark:border-gray-700">
								<div class="mb-2 flex items-center gap-1.5">
									<span class="h-2 w-2 rounded-full {meta.color}"></span>
									<span class="text-xs font-medium text-gray-600 dark:text-gray-400">{meta.label}</span>
									<span class="ml-auto text-xs text-gray-400">{users.length}</span>
								</div>
								{#if users.length === 0}
									<p class="text-xs text-gray-400 italic">—</p>
								{:else}
									<div class="flex flex-wrap gap-1">
										{#each users as u}
											<span class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-700 dark:bg-gray-700 dark:text-gray-300">{u}</span>
										{/each}
									</div>
								{/if}
							</div>
						{/each}
					</div>

					<!-- Permission editor -->
					<div class="rounded-lg bg-gray-50 p-4 dark:bg-gray-700/50">
						<h3 class="mb-3 text-sm font-medium text-gray-700 dark:text-gray-300">Update permissions</h3>
						<div class="space-y-3">
							<div class="flex gap-2">
								<select bind:value={permType} class="select-field flex-none w-40">
									{#each Object.entries(permLabels) as [type, meta]}
										<option value={type}>{meta.label}</option>
									{/each}
								</select>
								<div class="flex-1 text-xs text-gray-500 dark:text-gray-400 self-center">
									Replace the selected list with the users below (space/comma separated)
								</div>
							</div>
							<textarea
								bind:value={permInput}
								placeholder="alice bob IT\carol @Domain Users"
								rows="3"
								class="input-field w-full resize-none font-mono text-xs"
							></textarea>
							<button onclick={savePermissions}
								class="rounded bg-blue-600 px-4 py-1.5 text-sm text-white hover:bg-blue-700">
								Apply
							</button>
						</div>
					</div>
				</div>
			</div>
		{:else}
			<div class="flex h-64 items-center justify-center text-gray-400">
				Select a share to view and edit permissions
			</div>
		{/if}
	</div>
</div>

