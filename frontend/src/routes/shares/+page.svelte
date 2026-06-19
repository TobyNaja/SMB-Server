<script lang="ts">
	import { onMount } from 'svelte';
	import { sharesApi, type Share, type PermissionType } from '$lib/api/shares';
	import { usersApi } from '$lib/api/users';
	import { groupsApi } from '$lib/api/groups';
	import { toast, toastError } from '$lib/stores/toast.svelte';
	import ConfirmModal from '$lib/components/ConfirmModal.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import { Folder, FolderOpen, Plus, Search, X, HelpCircle } from 'lucide-svelte';

	let shares     = $state<Share[]>([]);
	let loading    = $state(true);
	let selected   = $state<Share | null>(null);
	let showCreate = $state(false);
	let search     = $state('');

	// Create form
	let newName       = $state('');
	let newPath       = $state('');
	let newComment    = $state('');
	let newBrowseable = $state(true);
	let newGuestOk    = $state(false);

	// Permission editor
	let permInput = $state('');
	let permType  = $state<PermissionType>('valid_users');

	// Auto-suggest sources
	let suggestUsers  = $state<string[]>([]);
	let suggestGroups = $state<string[]>([]);
	let suggestLoaded = $state(false);

	// Help modal
	let showHelp = $state(false);

	// Autocomplete dropdown state
	let selectedIdx = $state(-1);
	let isFocused   = $state(false);
	let dropdownEl  = $state<HTMLDivElement | null>(null);

	// Confirm delete
	let confirmOpen = $state(false);
	let confirmName = $state('');

	const permLabels: Record<PermissionType, { label: string; dot: string; desc: string }> = {
		valid_users:   { label: 'Valid Users',   dot: 'bg-gray-400',   desc: 'Users allowed to connect' },
		write_list:    { label: 'Write List',    dot: 'bg-gcp-green',  desc: 'Can read & write files' },
		read_list:     { label: 'Read List',     dot: 'bg-gcp-blue',   desc: 'Can read files only' },
		admin_users:   { label: 'Admin Users',   dot: 'bg-purple-600', desc: 'Full admin access, bypass ACL' },
		invalid_users: { label: 'Blocked',       dot: 'bg-gcp-red',    desc: 'Explicitly denied (highest priority)' },
	};

	// Sidebar pagination
	let page = $state(1);
	const PAGE_SIZE = 15;

	$effect(() => { search; page = 1; });

	const filtered = $derived.by(() => {
		if (!search.trim()) return shares;
		const q = search.toLowerCase();
		return shares.filter(s =>
			s.name.toLowerCase().includes(q) ||
			(s.path ?? '').toLowerCase().includes(q) ||
			(s.comment ?? '').toLowerCase().includes(q)
		);
	});

	const paged = $derived(filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE));

	// Autocomplete: get last partial token the user is typing
	const lastToken = $derived.by(() => {
		const parts = permInput.split(/[\s,]+/);
		return parts[parts.length - 1].toLowerCase().trim();
	});

	// Already-entered values (so we don't suggest duplicates)
	const enteredValues = $derived(
		new Set(permInput.split(/[\s,]+/).map(s => s.trim().toLowerCase()).filter(Boolean))
	);

	const suggestions = $derived.by(() => {
		const tok = lastToken;
		const entered = enteredValues;

		const users = suggestUsers.filter(u => {
			const low = u.toLowerCase();
			return !entered.has(low) && (tok === '' || low.startsWith(tok));
		}).slice(0, 8);

		// Groups shown as @groupname
		const groups = suggestGroups.filter(g => {
			const low = g.toLowerCase();
			const withAt = '@' + low;
			return !entered.has(withAt) && (tok === '' || low.startsWith(tok.replace(/^@/, '')) || withAt.startsWith(tok));
		}).slice(0, 6);

		return { users, groups, hasAny: users.length > 0 || groups.length > 0 };
	});

	// Flat list for keyboard navigation
	const suggestionList = $derived.by(() => [
		...suggestions.users.map(u => ({ label: u, value: u, type: 'user' as const })),
		...suggestions.groups.map(g => ({ label: '@' + g, value: '@' + g, type: 'group' as const })),
	]);

	const showDropdown = $derived(isFocused && suggestions.hasAny);

	// Reset selection when suggestion list changes
	$effect(() => { suggestionList; selectedIdx = -1; });

	// Scroll selected item into view
	$effect(() => {
		if (selectedIdx < 0 || !dropdownEl) return;
		const item = dropdownEl.children[selectedIdx] as HTMLElement | undefined;
		item?.scrollIntoView({ block: 'nearest' });
	});

	function handlePermKeydown(e: KeyboardEvent) {
		const list = suggestionList;
		if (!showDropdown || list.length === 0) return;

		if (e.key === 'ArrowDown') {
			e.preventDefault();
			selectedIdx = selectedIdx < list.length - 1 ? selectedIdx + 1 : 0;
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			selectedIdx = selectedIdx > 0 ? selectedIdx - 1 : list.length - 1;
		} else if ((e.key === 'Tab' || e.key === 'Enter') && selectedIdx >= 0) {
			e.preventDefault();
			addSuggestion(list[selectedIdx].value);
			selectedIdx = -1;
		} else if (e.key === 'Escape') {
			isFocused = false;
			selectedIdx = -1;
		}
	}

	async function load() {
		loading = true;
		try {
			const r = await sharesApi.list();
			shares = r.shares ?? [];
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to load shares');
		} finally {
			loading = false;
		}
	}

	async function loadSuggestions() {
		if (suggestLoaded) return;
		try {
			const [ur, gr] = await Promise.all([usersApi.list(), groupsApi.list()]);
			suggestUsers  = (ur.users ?? []).map(u => u.username);
			suggestGroups = gr.groups ?? [];
			suggestLoaded = true;
		} catch { /* suggestions are a nice-to-have */ }
	}

	function addSuggestion(value: string) {
		const parts = permInput.trim().split(/[\s,]+/).filter(Boolean);
		const tok = lastToken;
		// Replace partial last token if it's a prefix of the suggestion
		if (tok && value.toLowerCase().startsWith(tok.replace(/^@/, '')) && parts.length > 0) {
			parts[parts.length - 1] = value;
		} else if (!parts.map(p => p.toLowerCase()).includes(value.toLowerCase())) {
			parts.push(value);
		}
		permInput = parts.join(' ') + ' ';
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

	function askDelete(name: string) { confirmName = name; confirmOpen = true; }

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
			loadSuggestions(); // lazy load on first select
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
			permInput = '';
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
			invalid_users: share.invalid_users,
		};
		return map[type] ?? [];
	}

	onMount(load);
</script>

<!-- Confirm delete -->
<ConfirmModal
	open={confirmOpen}
	title="Delete share"
	message="Delete share '{confirmName}'? This removes it from Samba configuration but does NOT delete files on disk."
	confirmLabel="Delete"
	danger={true}
	onconfirm={deleteShare}
	oncancel={() => (confirmOpen = false)}
/>

<!-- Help modal -->
{#if showHelp}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4"
		onclick={(e) => { if (e.target === e.currentTarget) showHelp = false; }}>
		<div class="card w-full max-w-lg overflow-y-auto shadow-lg" style="max-height: 90vh">
			<div class="flex items-center justify-between border-b border-gcp-border px-5 py-4">
				<h2 class="text-sm font-semibold text-gcp-dark">Permission Guide</h2>
				<button onclick={() => (showHelp = false)} class="text-gcp-muted hover:text-gcp-dark">
					<X size={16} />
				</button>
			</div>

			<div class="space-y-5 px-5 py-4 text-xs">
				<!-- Permission types -->
				<section>
					<h3 class="mb-2 font-semibold uppercase tracking-wide text-gcp-muted">Permission Lists</h3>
					<div class="space-y-2">
						{#each Object.entries(permLabels) as [type, meta]}
							<div class="flex gap-3 rounded border border-gcp-border p-3">
								<span class="mt-0.5 h-2 w-2 flex-none rounded-full {meta.dot}"></span>
								<div>
									<div class="font-medium text-gcp-dark">{meta.label}</div>
									<div class="text-gcp-muted">{meta.desc}</div>
								</div>
							</div>
						{/each}
					</div>
				</section>

				<!-- Priority rules -->
				<section>
					<h3 class="mb-2 font-semibold uppercase tracking-wide text-gcp-muted">Sync Rules (applied on every save)</h3>
					<ol class="space-y-1.5 text-gcp-muted list-decimal list-inside">
						<li><span class="font-mono text-gcp-red">invalid_users</span> evicts the user from all other lists</li>
						<li><span class="font-mono text-purple-700">admin_users</span> are removed from write/read list (admin supersedes them) and auto-added to valid_users</li>
						<li><span class="font-mono text-gcp-green">write_list</span> users are removed from read_list and auto-added to valid_users</li>
						<li><span class="font-mono text-gcp-blue">read_list</span> users are auto-added to valid_users</li>
					</ol>
					<p class="mt-2 text-gcp-muted">You only need to set one list at a time — the sync keeps everything consistent.</p>
				</section>

				<!-- Username formats -->
				<section>
					<h3 class="mb-2 font-semibold uppercase tracking-wide text-gcp-muted">Username Formats</h3>
					<div class="space-y-1.5">
						{#each [
							{ ex: 'alice',           desc: 'Local Samba user' },
							{ ex: 'IT\\username',    desc: 'Active Directory user (domain\\user)' },
							{ ex: '@groupname',      desc: 'Local Linux group' },
							{ ex: '@"Group Name"',   desc: 'AD group with spaces' },
						] as row}
							<div class="flex items-baseline gap-3 rounded bg-gcp-bg px-3 py-2">
								<code class="w-36 flex-none font-mono text-gcp-dark">{row.ex}</code>
								<span class="text-gcp-muted">{row.desc}</span>
							</div>
						{/each}
					</div>
					<p class="mt-2 text-gcp-muted">Separate multiple entries with a space or comma.</p>
				</section>
			</div>
		</div>
	</div>
{/if}

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

		<div class="relative mb-3">
			<Search size={13} class="absolute left-2.5 top-1/2 -translate-y-1/2 text-gcp-muted" />
			<input bind:value={search} placeholder="Search shares…" class="input-field w-full pl-8 text-xs" />
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
					<label class="flex cursor-pointer items-center gap-2 text-xs text-gcp-muted">
						<input type="checkbox" bind:checked={newBrowseable} class="rounded" /> Browseable
					</label>
					<label class="flex cursor-pointer items-center gap-2 text-xs text-gcp-muted">
						<input type="checkbox" bind:checked={newGuestOk} class="rounded" /> Guest OK
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
				{#each paged as share}
					<!-- svelte-ignore a11y_interactive_supports_focus -->
					<div
						role="button"
						tabindex="0"
						onclick={() => selectShare(share.name)}
						onkeypress={(e) => e.key === 'Enter' && selectShare(share.name)}
						class="group w-full cursor-pointer rounded px-3 py-2.5 text-left transition-colors
							{selected?.name === share.name
							? 'border-l-2 border-gcp-blue bg-gcp-blue-light'
							: 'border-l-2 border-transparent hover:bg-white'}"
					>
						<div class="flex items-center justify-between">
							<span class="flex items-center gap-1.5 text-sm font-medium
								{selected?.name === share.name ? 'text-gcp-blue' : 'text-gcp-dark'}">
								<Folder size={13} class="flex-none opacity-70" />{share.name}
							</span>
							<button
								onclick={(e) => { e.stopPropagation(); askDelete(share.name); }}
								class="rounded p-0.5 text-gcp-muted opacity-0 transition-opacity group-hover:opacity-100 hover:text-gcp-red"
							><X size={12} /></button>
						</div>
						<div class="mt-0.5 truncate text-xs text-gcp-muted">{share.path}</div>
					</div>
				{/each}
			</div>
			{#if filtered.length > PAGE_SIZE}
				<Pagination
					total={filtered.length}
					{page}
					pageSize={PAGE_SIZE}
					onPageChange={(p) => (page = p)}
				/>
			{/if}
		{/if}
	</div>

	<!-- Share detail -->
	<div class="min-w-0 flex-1">
		{#if selected}
			<div class="card">
				<div class="flex items-center gap-3 border-b border-gcp-border px-5 py-4">
					<FolderOpen size={18} class="text-gcp-blue flex-none" />
					<div class="min-w-0 flex-1">
						<h2 class="text-sm font-semibold text-gcp-dark">{selected.name}</h2>
						<p class="text-xs text-gcp-muted">{selected.path}</p>
					</div>
				</div>

				<div class="p-5">
					<!-- Property toggles -->
					<div class="mb-5 flex flex-wrap gap-2">
						<button onclick={() => toggleProp('read_only')} title="Click to toggle"
							class="badge cursor-pointer transition-colors
								{selected.read_only ? 'bg-yellow-100 text-yellow-800 hover:bg-yellow-200' : 'bg-green-100 text-gcp-green hover:bg-green-200'}">
							{selected.read_only ? 'Read-only' : 'Writable'}
						</button>
						<button onclick={() => toggleProp('browseable')} title="Click to toggle"
							class="badge cursor-pointer transition-colors
								{selected.browseable ? 'bg-gcp-blue-light text-gcp-blue hover:bg-blue-200' : 'bg-gray-100 text-gcp-muted hover:bg-gray-200'}">
							Browseable {selected.browseable ? 'on' : 'off'}
						</button>
						<button onclick={() => toggleProp('guest_ok')} title="Click to toggle"
							class="badge cursor-pointer transition-colors
								{selected.guest_ok ? 'bg-orange-100 text-orange-800 hover:bg-orange-200' : 'bg-gray-100 text-gcp-muted hover:bg-gray-200'}">
							Guest {selected.guest_ok ? 'allowed' : 'off'}
						</button>
						{#if selected.abse}<span class="badge bg-purple-100 text-purple-800">ABSE</span>{/if}
						{#if selected.comment}<span class="self-center text-xs text-gcp-muted">{selected.comment}</span>{/if}
					</div>

					<!-- Permission list display -->
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
									<p class="text-xs italic text-gcp-muted">—</p>
								{:else}
									<div class="flex flex-wrap gap-1">
										{#each users as u}
											<span class="rounded bg-gcp-bg px-1.5 py-0.5 font-mono text-xs text-gcp-dark">{u}</span>
										{/each}
									</div>
								{/if}
							</div>
						{/each}
					</div>

					<!-- Permission editor -->
					<div class="rounded border border-gcp-border bg-gcp-bg p-4">
						<!-- Header with help button -->
						<div class="mb-3 flex items-center gap-2">
							<h3 class="flex-1 text-xs font-semibold uppercase tracking-wide text-gcp-muted">Update permissions</h3>
							<button
								onclick={() => (showHelp = true)}
								title="How permissions work"
								class="flex items-center gap-1 rounded border border-gcp-border bg-white px-2 py-1
									text-xs text-gcp-muted hover:border-gcp-blue hover:text-gcp-blue transition-colors"
							>
								<HelpCircle size={12} /> How it works
							</button>
						</div>

						<div class="space-y-3">
							<!-- Type selector -->
							<select bind:value={permType} class="select-field w-44">
								{#each Object.entries(permLabels) as [type, meta]}
									<option value={type}>{meta.label}</option>
								{/each}
							</select>

							<!-- Textarea with IDE-like autocomplete -->
							<div class="relative">
								<textarea
									bind:value={permInput}
									placeholder="alice  IT\carol  @admins"
									rows="2"
									class="input-field w-full resize-none font-mono text-xs"
									onkeydown={handlePermKeydown}
									onfocus={() => (isFocused = true)}
									onblur={() => setTimeout(() => { isFocused = false; selectedIdx = -1; }, 150)}
								></textarea>

								{#if showDropdown}
									<div
										bind:this={dropdownEl}
										class="absolute left-0 top-full z-20 mt-0.5 w-full overflow-y-auto rounded border border-gcp-border bg-white shadow-lg"
										style="max-height: 192px"
									>
										{#each suggestionList as item, i}
											<button
												type="button"
												onmousedown={(e) => { e.preventDefault(); addSuggestion(item.value); selectedIdx = -1; }}
												class="flex w-full items-center gap-2 px-3 py-1.5 text-left font-mono text-xs transition-colors
													{i === selectedIdx
														? 'bg-gcp-blue text-white'
														: item.type === 'group'
															? 'text-purple-700 hover:bg-purple-50'
															: 'text-gcp-dark hover:bg-gcp-bg'}"
											>
												<span class="flex-1">{item.label}</span>
												<span class="font-sans text-[10px] opacity-60">{item.type}</span>
											</button>
										{/each}
									</div>
								{/if}
							</div>

							<div class="flex items-center gap-3">
								<button onclick={savePermissions} class="btn-primary text-xs px-3 py-1.5">Apply</button>
								{#if permInput.trim()}
									<button onclick={() => (permInput = '')}
										class="text-xs text-gcp-muted hover:text-gcp-dark transition-colors">Clear</button>
								{/if}
							</div>
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
