<script lang="ts">
	import { onMount } from 'svelte';
	import { sharesApi, type Share, type PermissionType } from '$lib/api/shares';
	import { toast, toastError } from '$lib/stores/toast.svelte';
	import ConfirmModal from '$lib/components/ConfirmModal.svelte';
	import { Folder, FolderOpen, Plus, Search, X } from 'lucide-svelte';

	let shares   = $state<Share[]>([]);
	let loading  = $state(true);
	let selected = $state<Share | null>(null);
	let showCreate = $state(false);
	let search   = $state('');

	// Create form
	let newName      = $state('');
	let newPath      = $state('');
	let newComment   = $state('');
	let newBrowseable = $state(true);
	let newGuestOk   = $state(false);

	// Permission editor
	let permInput = $state('');
	let permType  = $state<PermissionType>('valid_users');

	// Confirm delete
	let confirmOpen  = $state(false);
	let confirmName  = $state('');

	const permLabels: Record<PermissionType, { label: string; dot: string }> = {
		valid_users:   { label: 'Valid Users',   dot: 'bg-gray-400'     },
		write_list:    { label: 'Write List',    dot: 'bg-gcp-green'    },
		read_list:     { label: 'Read List',     dot: 'bg-gcp-blue'     },
		admin_users:   { label: 'Admin Users',   dot: 'bg-purple-600'   },
		invalid_users: { label: 'Blocked',       dot: 'bg-gcp-red'      }
	};

	const filtered = $derived(
		search.trim()
			? shares.filter(s => {
				const q = search.toLowerCase();
				return (
					s.name.toLowerCase().includes(q) ||
					(s.path ?? '').toLowerCase().includes(q) ||
					(s.comment ?? '').toLowerCase().includes(q)
				);
			})
			: shares
	);

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

	function askDelete(name: string) {
		confirmName = name;
		confirmOpen = true;
	}

	async function deleteShare() {
		confirmOpen = false;
		try {
			await sharesApi.delete(confirmName);
			toast(`Share '${confirmName}' deleted`);
			if (selected?.name === confirmName) selected = null;
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

	async function toggleProp(prop: 'browseable' | 'read_only' | 'guest_ok') {
		if (!selected) return;
		const newVal = !selected[prop];
		try {
			await sharesApi.update(selected.name, { [prop]: newVal });
			selected = { ...selected, [prop]: newVal };
			// sync in list
			shares = shares.map(s => s.name === selected!.name ? { ...s, [prop]: newVal } : s);
			toast(`${prop.replace('_', ' ')} set to ${newVal}`);
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to update share');
		}
	}

	function getUserList(share: Share, type: PermissionType): string[] {
		const map: Record<PermissionType, string[] | null | undefined> = {
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

<ConfirmModal
	open={confirmOpen}
	title="Delete share"
	message="Delete share '{confirmName}'? This removes it from Samba configuration but does NOT delete files on disk."
	confirmLabel="Delete"
	danger={true}
	onconfirm={deleteShare}
	oncancel={() => (confirmOpen = false)}
/>

<div class="flex h-full gap-5">
	<!-- Share list panel -->
	<div class="w-64 flex-none">
		<div class="mb-3 flex items-center gap-2">
			<h1 class="page-title flex-1">Shares</h1>
			<button
				onclick={() => (showCreate = !showCreate)}
				class="flex items-center gap-1 rounded border border-gcp-border bg-white px-2.5 py-1.5
					text-xs text-gcp-dark hover:bg-gcp-bg transition-colors"
			>
				<Plus size={12} />New
			</button>
		</div>

		<!-- Search -->
		<div class="relative mb-3">
			<Search size={13} class="absolute left-2.5 top-1/2 -translate-y-1/2 text-gcp-muted" />
			<input
				bind:value={search}
				placeholder="Search shares…"
				class="input-field w-full pl-8 text-xs"
			/>
			{#if search}
				<button onclick={() => (search = '')} class="absolute right-2 top-1/2 -translate-y-1/2 text-gcp-muted hover:text-gcp-dark">
					<X size={12} />
				</button>
			{/if}
		</div>

		{#if showCreate}
			<div class="card mb-3 p-4">
				<h2 class="mb-3 text-xs font-semibold uppercase tracking-wide text-gcp-muted">New Share</h2>
				<form onsubmit={(e) => { e.preventDefault(); createShare(); }} class="space-y-2">
					<input bind:value={newName} placeholder="Share name" required class="input-field w-full" />
					<input bind:value={newPath} placeholder="/srv/shares/name" required class="input-field w-full" />
					<input bind:value={newComment} placeholder="Description (optional)" class="input-field w-full" />
					<label class="flex items-center gap-2 text-xs text-gcp-muted cursor-pointer">
						<input type="checkbox" bind:checked={newBrowseable} class="rounded" />
						Browseable
					</label>
					<label class="flex items-center gap-2 text-xs text-gcp-muted cursor-pointer">
						<input type="checkbox" bind:checked={newGuestOk} class="rounded" />
						Guest OK
					</label>
					<div class="flex gap-2 pt-1">
						<button type="submit" class="btn-primary flex-1 py-1.5 text-xs">Create</button>
						<button type="button" onclick={() => (showCreate = false)} class="btn-secondary flex-1 py-1.5 text-xs">Cancel</button>
					</div>
				</form>
			</div>
		{/if}

		{#if loading}
			<div class="text-xs text-gcp-muted">Loading…</div>
		{:else if filtered.length === 0}
			<div class="text-xs text-gcp-muted">{search ? 'No matches' : 'No shares found'}</div>
		{:else}
			<div class="space-y-0.5">
				{#each filtered as share}
					<!-- svelte-ignore a11y_interactive_supports_focus -->
					<div
						role="button"
						tabindex="0"
						onclick={() => selectShare(share.name)}
						onkeypress={(e) => e.key === 'Enter' && selectShare(share.name)}
						class="group w-full cursor-pointer rounded px-3 py-2.5 text-left transition-colors
							{selected?.name === share.name
							? 'bg-gcp-blue-light border-l-2 border-gcp-blue'
							: 'border-l-2 border-transparent hover:bg-white'}"
					>
						<div class="flex items-center justify-between">
							<span class="flex items-center gap-1.5 text-sm font-medium
								{selected?.name === share.name ? 'text-gcp-blue' : 'text-gcp-dark'}">
								<Folder size={13} class="flex-none opacity-70" />{share.name}
							</span>
							<button
								onclick={(e) => { e.stopPropagation(); askDelete(share.name); }}
								class="rounded p-0.5 text-gcp-muted opacity-0 group-hover:opacity-100 hover:text-gcp-red transition-opacity"
							><X size={12} /></button>
						</div>
						<div class="mt-0.5 truncate text-xs text-gcp-muted">{share.path}</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Share detail -->
	<div class="min-w-0 flex-1">
		{#if selected}
			<div class="card">
				<!-- Header -->
				<div class="flex items-center gap-3 border-b border-gcp-border px-5 py-4">
					<FolderOpen size={18} class="text-gcp-blue flex-none" />
					<div class="min-w-0 flex-1">
						<h2 class="text-sm font-semibold text-gcp-dark">{selected.name}</h2>
						<p class="text-xs text-gcp-muted">{selected.path}</p>
					</div>
				</div>

				<div class="p-5">
					<!-- Clickable property toggles -->
					<div class="mb-5 flex flex-wrap gap-2">
						<button
							onclick={() => toggleProp('read_only')}
							title="Click to toggle"
							class="badge cursor-pointer transition-colors
								{selected.read_only
								? 'bg-yellow-100 text-yellow-800 hover:bg-yellow-200'
								: 'bg-green-100 text-gcp-green hover:bg-green-200'}"
						>
							{selected.read_only ? 'Read-only' : 'Writable'}
						</button>
						<button
							onclick={() => toggleProp('browseable')}
							title="Click to toggle"
							class="badge cursor-pointer transition-colors
								{selected.browseable
								? 'bg-gcp-blue-light text-gcp-blue hover:bg-blue-200'
								: 'bg-gray-100 text-gcp-muted hover:bg-gray-200'}"
						>
							Browseable {selected.browseable ? 'on' : 'off'}
						</button>
						<button
							onclick={() => toggleProp('guest_ok')}
							title="Click to toggle"
							class="badge cursor-pointer transition-colors
								{selected.guest_ok
								? 'bg-orange-100 text-orange-800 hover:bg-orange-200'
								: 'bg-gray-100 text-gcp-muted hover:bg-gray-200'}"
						>
							Guest {selected.guest_ok ? 'allowed' : 'off'}
						</button>
						{#if selected.abse}
							<span class="badge bg-purple-100 text-purple-800">ABSE</span>
						{/if}
						{#if selected.comment}
							<span class="text-xs text-gcp-muted self-center">{selected.comment}</span>
						{/if}
					</div>

					<!-- User permission lists -->
					<div class="mb-5 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
						{#each Object.entries(permLabels) as [type, meta]}
							{@const users = getUserList(selected, type as PermissionType)}
							<div class="rounded border border-gcp-border p-3">
								<div class="mb-2 flex items-center gap-1.5">
									<span class="h-2 w-2 rounded-full {meta.dot}"></span>
									<span class="text-xs font-medium text-gcp-muted">{meta.label}</span>
									<span class="ml-auto text-xs text-gcp-muted">{users.length}</span>
								</div>
								{#if users.length === 0}
									<p class="text-xs text-gcp-muted italic">—</p>
								{:else}
									<div class="flex flex-wrap gap-1">
										{#each users as u}
											<span class="rounded bg-gcp-bg px-1.5 py-0.5 text-xs text-gcp-dark font-mono">{u}</span>
										{/each}
									</div>
								{/if}
							</div>
						{/each}
					</div>

					<!-- Permission editor -->
					<div class="rounded border border-gcp-border bg-gcp-bg p-4">
						<h3 class="mb-3 text-xs font-semibold uppercase tracking-wide text-gcp-muted">Update permissions</h3>
						<div class="space-y-3">
							<div class="flex gap-2">
								<select bind:value={permType} class="select-field w-40 flex-none">
									{#each Object.entries(permLabels) as [type, meta]}
										<option value={type}>{meta.label}</option>
									{/each}
								</select>
								<p class="self-center text-xs text-gcp-muted">
									Replace the list — space or comma separated
								</p>
							</div>
							<textarea
								bind:value={permInput}
								placeholder="alice bob IT\carol @Domain Users"
								rows="3"
								class="input-field w-full resize-none font-mono text-xs"
							></textarea>
							<button onclick={savePermissions} class="btn-primary text-xs px-3 py-1.5">Apply</button>
						</div>
					</div>
				</div>
			</div>
		{:else}
			<div class="flex h-48 items-center justify-center rounded border border-dashed border-gcp-border text-sm text-gcp-muted">
				Select a share to view and edit
			</div>
		{/if}
	</div>
</div>
