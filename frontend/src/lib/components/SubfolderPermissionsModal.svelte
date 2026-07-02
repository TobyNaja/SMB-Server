<script lang="ts">
	import { untrack } from 'svelte';
	import { sharesApi, type Share, type SubfolderAclEntry, type SubfolderPerm } from '$lib/api/shares';
	import { usersApi } from '$lib/api/users';
	import { toast, toastError } from '$lib/stores/toast.svelte';
	import { X, FolderTree, RefreshCw, Lock, LockOpen, ShieldAlert } from 'lucide-svelte';

	interface Props {
		open: boolean;
		share: Share | null;
		onclose: () => void;
	}
	let { open, share, onclose }: Props = $props();

	let path       = $state('');
	let entries    = $state<SubfolderAclEntry[]>([]);
	let loading    = $state(false);
	let loadedPath = $state<string | null>(null);
	let locked     = $state(false);

	// Grant / update form
	let username  = $state('');
	let perms     = $state<SubfolderPerm>('rx');
	let recursive = $state(false);
	let saving    = $state(false);

	// Lock (make private) form
	let lockUsers     = $state('');
	let lockPerms     = $state<SubfolderPerm>('rx');
	let lockRecursive = $state(false);
	let busyLock      = $state(false);

	// Datalist source (local Samba users). AD users can be typed as DOMAIN\user.
	let localUsers = $state<string[]>([]);

	const permOptions: { value: SubfolderPerm; label: string }[] = [
		{ value: 'r',   label: 'Read only (r)' },
		{ value: 'rx',  label: 'Read & traverse (rx)' },
		{ value: 'rwx', label: 'Full access (rwx)' },
	];

	// A directory needs traverse (x) to be enterable, so the lock form omits "r"
	// — granting "r" only would shut out even the allowlisted users.
	const lockPermOptions = permOptions.filter((o) => o.value !== 'r');

	// Reset + load whenever the modal opens for a share. untrack() keeps the
	// effect from re-firing on path/entries edits — it only watches open+share.
	$effect(() => {
		if (open && share) {
			untrack(() => {
				path = '';
				entries = [];
				loadedPath = null;
				locked = false;
				username = '';
				perms = 'rx';
				recursive = false;
				lockUsers = '';
				lockPerms = 'rx';
				lockRecursive = false;
				void loadUsers();
				void loadEntries();
			});
		}
	});

	async function loadUsers() {
		if (localUsers.length > 0) return;
		try {
			const r = await usersApi.list();
			localUsers = (r.users ?? []).map((u) => u.username);
		} catch {
			/* suggestions are optional */
		}
	}

	async function loadEntries() {
		if (!share) return;
		loading = true;
		try {
			const r = await sharesApi.getSubfolderPermissions(share.name, path);
			entries = r.entries ?? [];
			loadedPath = r.path;
			locked = r.locked;
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to read permissions');
			entries = [];
			loadedPath = null;
		} finally {
			loading = false;
		}
	}

	async function grant() {
		if (!share) return;
		const u = username.trim();
		if (!u) return;
		saving = true;
		try {
			await sharesApi.setSubfolderPermission(share.name, {
				subfolder_path: path,
				username: u,
				permissions: perms,
				recursive,
			});
			toast(`Granted ${perms} to ${u}`);
			username = '';
			await loadEntries();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to update permissions');
		} finally {
			saving = false;
		}
	}

	async function revoke(name: string) {
		if (!share) return;
		try {
			await sharesApi.setSubfolderPermission(share.name, {
				subfolder_path: path,
				username: name,
				permissions: 'none',
				recursive,
			});
			toast(`Revoked ${name}`);
			await loadEntries();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to revoke');
		}
	}

	// Parsed allowlist for the lock form.
	const lockUsersArr = $derived(
		lockUsers.split(/[\s,]+/).map((s) => s.trim()).filter(Boolean)
	);

	// Allowlisted users who can't even connect to the share (not in valid users)
	// — locking them in is pointless until they're added at the share level.
	const notValidUsers = $derived.by(() => {
		const valid = new Set((share?.valid_users ?? []).map((v) => v.toLowerCase()));
		return lockUsersArr.filter((u) => !valid.has(u.toLowerCase()));
	});

	// The share root can't be locked (backend refuses) — only real subfolders.
	const canLock = $derived(loadedPath !== null && loadedPath !== '.');

	async function doLock() {
		if (!share || !canLock) return;
		busyLock = true;
		try {
			await sharesApi.lockSubfolder(share.name, {
				subfolder_path: path,
				users: lockUsersArr,
				permissions: lockPerms,
				recursive: lockRecursive,
			});
			toast(`Locked "${loadedPath}" to ${lockUsersArr.length} user(s)`);
			lockUsers = '';
			await loadEntries();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to lock folder');
		} finally {
			busyLock = false;
		}
	}

	async function doUnlock() {
		if (!share) return;
		busyLock = true;
		try {
			await sharesApi.unlockSubfolder(share.name, { subfolder_path: path, recursive: lockRecursive });
			toast(`Unlocked "${loadedPath}"`);
			await loadEntries();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to unlock folder');
		} finally {
			busyLock = false;
		}
	}

	// A named-user access entry is the only kind we can revoke/overwrite here;
	// owner/group rows (empty name) and inherited default rows are read-only.
	const canRevoke = (e: SubfolderAclEntry) => e.type === 'user' && e.name !== '' && !e.default;

	function displayName(e: SubfolderAclEntry): string {
		if (e.name !== '') return e.name;
		return e.type === 'user' ? '(owner)' : '(owning group)';
	}
</script>

{#if open && share}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4"
		onclick={(e) => { if (e.target === e.currentTarget) onclose(); }}
	>
		<div class="card flex w-full max-w-2xl flex-col overflow-hidden shadow-lg" style="max-height: 90vh">
			<!-- Header -->
			<div class="flex items-center gap-2 border-b border-gcp-border px-5 py-4">
				<FolderTree size={18} class="flex-none text-gcp-blue" />
				<div class="min-w-0 flex-1">
					<h2 class="text-sm font-semibold text-gcp-dark">Subfolder Permissions</h2>
					<p class="truncate text-xs text-gcp-muted">{share.name} · {share.path}</p>
				</div>
				<button onclick={onclose} class="text-gcp-muted hover:text-gcp-dark"><X size={16} /></button>
			</div>

			<div class="space-y-5 overflow-y-auto px-5 py-4">
				<!-- Path selector -->
				<div>
					<label for="sub-path" class="mb-1 block text-xs font-semibold uppercase tracking-wide text-gcp-muted">
						Subfolder (relative to share root)
					</label>
					<div class="flex gap-2">
						<input
							id="sub-path"
							bind:value={path}
							placeholder="e.g. Secret_Plan  (empty = share root)"
							class="input-field w-full font-mono text-xs"
							onkeydown={(e) => e.key === 'Enter' && loadEntries()}
						/>
						<button onclick={loadEntries} disabled={loading} class="btn-secondary flex items-center gap-1 px-3 py-1.5 text-xs">
							<RefreshCw size={12} class={loading ? 'animate-spin' : ''} /> Load
						</button>
					</div>
					{#if loadedPath}
						<div class="mt-2 flex items-center gap-2 text-xs">
							<span class="text-gcp-muted">Showing ACLs for <span class="font-mono">{loadedPath}</span></span>
							{#if locked}
								<span class="badge inline-flex items-center gap-1 bg-red-50 text-gcp-red"><Lock size={11} /> Private</span>
							{:else}
								<span class="badge inline-flex items-center gap-1 bg-green-100 text-gcp-green"><LockOpen size={11} /> Open</span>
							{/if}
						</div>
					{/if}
				</div>

				<!-- Current ACL entries -->
				<div>
					<h3 class="mb-2 text-xs font-semibold uppercase tracking-wide text-gcp-muted">Current entries</h3>
					{#if loading}
						<p class="text-xs text-gcp-muted">Loading…</p>
					{:else if entries.length === 0}
						<p class="text-xs italic text-gcp-muted">No ACL entries (or path not loaded).</p>
					{:else}
						<div class="overflow-hidden rounded border border-gcp-border">
							<table class="w-full text-xs">
								<thead class="bg-gcp-bg text-gcp-muted">
									<tr>
										<th class="px-3 py-1.5 text-left font-medium">Kind</th>
										<th class="px-3 py-1.5 text-left font-medium">Name</th>
										<th class="px-3 py-1.5 text-left font-medium">Perms</th>
										<th class="px-3 py-1.5 text-left font-medium">Scope</th>
										<th class="px-3 py-1.5"></th>
									</tr>
								</thead>
								<tbody>
									{#each entries as e (e.type + ':' + e.name + ':' + e.default)}
										<tr class="border-t border-gcp-border">
											<td class="px-3 py-1.5 text-gcp-muted">{e.type}</td>
											<td class="px-3 py-1.5 font-mono text-gcp-dark">{displayName(e)}</td>
											<td class="px-3 py-1.5 font-mono text-gcp-dark">{e.perms}</td>
											<td class="px-3 py-1.5">
												{#if e.default}
													<span class="badge bg-gcp-blue-light text-gcp-blue">inherited</span>
												{:else}
													<span class="text-gcp-muted">this folder</span>
												{/if}
											</td>
											<td class="px-3 py-1.5 text-right">
												{#if canRevoke(e)}
													<button
														onclick={() => revoke(e.name)}
														title="Revoke {e.name}"
														class="rounded p-0.5 text-gcp-muted transition-colors hover:bg-red-50 hover:text-gcp-red"
													><X size={12} /></button>
												{/if}
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{/if}
				</div>

				<!-- Private lock / unlock -->
				{#if loadedPath}
					<div class="rounded border p-4 {locked ? 'border-gcp-red/40 bg-red-50/40' : 'border-gcp-border bg-gcp-bg'}">
						{#if locked}
							<div class="flex flex-wrap items-center gap-3">
								<div class="flex items-center gap-2 text-xs font-semibold text-gcp-red">
									<Lock size={14} /> This folder is private
								</div>
								<p class="flex-1 text-xs text-gcp-muted">
									Only the users listed above can enter — everyone else is blocked and can't see it.
								</p>
								<label class="flex cursor-pointer items-center gap-2 text-xs text-gcp-muted" title="Apply to existing contents too">
									<input type="checkbox" bind:checked={lockRecursive} class="rounded" /> Recursive
								</label>
								<button onclick={doUnlock} disabled={busyLock} class="btn-secondary flex items-center gap-1 px-3 py-1.5 text-xs">
									<LockOpen size={12} /> {busyLock ? 'Unlocking…' : 'Unlock'}
								</button>
							</div>
						{:else}
							<h3 class="mb-3 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-gcp-muted">
								<Lock size={12} /> Make private (lock)
							</h3>
							{#if !canLock}
								<p class="text-xs italic text-gcp-muted">The share root can't be locked — load a specific subfolder first.</p>
							{:else}
								<div class="flex flex-wrap items-end gap-3">
									<div class="min-w-[12rem] flex-1">
										<label for="lock-users" class="mb-1 block text-xs text-gcp-muted">Allowed users (space/comma separated)</label>
										<input
											id="lock-users"
											bind:value={lockUsers}
											list="sub-user-list"
											placeholder="toby  IT\alice"
											class="input-field w-full font-mono text-xs"
										/>
									</div>
									<div>
										<label for="lock-perms" class="mb-1 block text-xs text-gcp-muted">Access</label>
										<select id="lock-perms" bind:value={lockPerms} class="select-field w-44 text-xs">
											{#each lockPermOptions as opt (opt.value)}<option value={opt.value}>{opt.label}</option>{/each}
										</select>
									</div>
									<label class="flex cursor-pointer items-center gap-2 py-1.5 text-xs text-gcp-muted" title="Apply to existing contents too">
										<input type="checkbox" bind:checked={lockRecursive} class="rounded" /> Recursive
									</label>
									<button onclick={doLock} disabled={busyLock} class="btn-danger flex items-center gap-1 px-3 py-1.5 text-xs">
										<Lock size={12} /> {busyLock ? 'Locking…' : 'Lock'}
									</button>
								</div>
								{#if notValidUsers.length > 0}
									<p class="mt-3 flex items-start gap-1.5 text-xs text-yellow-700">
										<ShieldAlert size={13} class="mt-0.5 flex-none" />
										<span>
											<span class="font-mono">{notValidUsers.join(', ')}</span> — not in this share's
											<span class="font-mono">valid users</span>, so they can't connect to the share at all. Add them at the share level first.
										</span>
									</p>
								{/if}
								<p class="mt-2 text-xs text-gcp-muted">
									Locking wipes inherited access and shuts out everyone not listed — they lose access <em>and</em> can no longer see the folder.
								</p>
							{/if}
						{/if}
					</div>
				{/if}

				<!-- Grant / update -->
				<div class="rounded border border-gcp-border bg-gcp-bg p-4">
					<h3 class="mb-3 text-xs font-semibold uppercase tracking-wide text-gcp-muted">Grant / update a user</h3>
					<div class="flex flex-wrap items-end gap-3">
						<div class="min-w-[10rem] flex-1">
							<label for="sub-user" class="mb-1 block text-xs text-gcp-muted">User (or DOMAIN\user)</label>
							<input
								id="sub-user"
								bind:value={username}
								list="sub-user-list"
								placeholder="alice  or  IT\carol"
								class="input-field w-full font-mono text-xs"
								onkeydown={(e) => e.key === 'Enter' && grant()}
							/>
							<datalist id="sub-user-list">
								{#each localUsers as u (u)}<option value={u}></option>{/each}
							</datalist>
						</div>
						<div>
							<label for="sub-perms" class="mb-1 block text-xs text-gcp-muted">Access</label>
							<select id="sub-perms" bind:value={perms} class="select-field w-44 text-xs">
								{#each permOptions as opt (opt.value)}<option value={opt.value}>{opt.label}</option>{/each}
							</select>
						</div>
						<label class="flex cursor-pointer items-center gap-2 py-1.5 text-xs text-gcp-muted" title="Apply to all existing files & subfolders">
							<input type="checkbox" bind:checked={recursive} class="rounded" /> Recursive
						</label>
						<button onclick={grant} disabled={saving || !username.trim()} class="btn-primary px-3 py-1.5 text-xs">
							{saving ? 'Applying…' : 'Apply'}
						</button>
					</div>
					<p class="mt-3 text-xs text-gcp-muted">
						POSIX ACLs are grant-only — this adds/updates access for one user. To block everyone except a
						chosen few, use <span class="font-medium text-gcp-red">Make private (lock)</span> above.
						The <span class="font-mono">Recursive</span> option also applies to revoke.
					</p>
				</div>
			</div>
		</div>
	</div>
{/if}
