<script lang="ts">
	import { onMount } from 'svelte';
	import { authApi, type AdminInfo } from '$lib/api/auth';
	import { getUser } from '$lib/stores/auth.svelte';
	import { toast, toastError } from '$lib/stores/toast.svelte';
	import ConfirmModal from '$lib/components/ConfirmModal.svelte';
	import { KeyRound, Plus } from 'lucide-svelte';

	let admins   = $state<AdminInfo[]>([]);
	let loading  = $state(true);
	let currentUser = $derived(getUser());

	// Change password form
	let oldPw = $state('');
	let newPw = $state('');
	let confirmPw = $state('');
	let pwLoading = $state(false);

	// Add admin form
	let newAdminUser = $state('');
	let newAdminPw   = $state('');
	let addLoading   = $state(false);

	// Remove admin confirm
	let confirmOpen = $state(false);
	let confirmAdmin = $state('');

	async function loadAdmins() {
		loading = true;
		try {
			const r = await authApi.listAdmins();
			admins = r.admins ?? [];
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to load admins');
		} finally {
			loading = false;
		}
	}

	async function changePassword(e: Event) {
		e.preventDefault();
		if (newPw !== confirmPw) { toastError('Passwords do not match'); return; }
		pwLoading = true;
		try {
			await authApi.changePassword(oldPw, newPw);
			toast('Password changed successfully');
			oldPw = ''; newPw = ''; confirmPw = '';
		} catch (err) {
			toastError(err instanceof Error ? err.message : 'Failed to change password');
		} finally {
			pwLoading = false;
		}
	}

	async function addAdmin(e: Event) {
		e.preventDefault();
		addLoading = true;
		try {
			await authApi.addAdmin(newAdminUser, newAdminPw);
			toast(`Admin '${newAdminUser}' created`);
			newAdminUser = ''; newAdminPw = '';
			await loadAdmins();
		} catch (err) {
			toastError(err instanceof Error ? err.message : 'Failed to create admin');
		} finally {
			addLoading = false;
		}
	}

	function askDelete(username: string) {
		confirmAdmin = username;
		confirmOpen  = true;
	}

	async function deleteAdmin() {
		confirmOpen = false;
		try {
			await authApi.deleteAdmin(confirmAdmin);
			toast(`Admin '${confirmAdmin}' removed`);
			await loadAdmins();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Failed to remove admin');
		}
	}

	function formatDate(ts: string) {
		return new Date(ts).toLocaleString('th-TH', { dateStyle: 'medium', timeStyle: 'short' });
	}

	onMount(loadAdmins);
</script>

<ConfirmModal
	open={confirmOpen}
	title="Remove administrator"
	message="Remove admin account '{confirmAdmin}'? This cannot be undone."
	confirmLabel="Remove"
	danger={true}
	onconfirm={deleteAdmin}
	oncancel={() => (confirmOpen = false)}
/>

<div class="max-w-2xl space-y-5">
	<h1 class="page-title">Settings</h1>

	<!-- Change password -->
	<div class="card p-5">
		<h2 class="mb-4 text-xs font-semibold uppercase tracking-wide text-gcp-muted">Change Your Password</h2>
		<form onsubmit={changePassword} class="space-y-3">
			<div>
				<label for="old-pw" class="mb-1 block text-xs text-gcp-muted">Current password</label>
				<input id="old-pw" type="password" bind:value={oldPw} required
					class="input-field w-full" placeholder="••••••••" autocomplete="current-password" />
			</div>
			<div>
				<label for="new-pw" class="mb-1 block text-xs text-gcp-muted">New password (min 8 characters)</label>
				<input id="new-pw" type="password" bind:value={newPw} required minlength="8"
					class="input-field w-full" placeholder="••••••••" autocomplete="new-password" />
			</div>
			<div>
				<label for="confirm-pw" class="mb-1 block text-xs text-gcp-muted">Confirm new password</label>
				<input id="confirm-pw" type="password" bind:value={confirmPw} required
					class="input-field w-full" placeholder="••••••••" autocomplete="new-password" />
			</div>
			<button type="submit" disabled={pwLoading} class="btn-primary text-xs py-1.5">
				{pwLoading ? 'Updating…' : 'Update Password'}
			</button>
		</form>
	</div>

	<!-- Administrators -->
	<div class="card p-5">
		<h2 class="mb-4 text-xs font-semibold uppercase tracking-wide text-gcp-muted">Administrators</h2>

		{#if loading}
			<div class="mb-4 text-sm text-gcp-muted">Loading…</div>
		{:else}
			<div class="mb-4 divide-y divide-gcp-border rounded border border-gcp-border">
				{#each admins as admin}
					<div class="flex items-center justify-between px-4 py-3">
						<div>
							<div class="flex items-center gap-2 text-sm font-medium text-gcp-dark">
								<KeyRound size={13} class="flex-none text-gcp-muted" />
								{admin.username}
								{#if admin.username === currentUser?.username}
									<span class="badge bg-gcp-blue-light text-gcp-blue">you</span>
								{/if}
							</div>
							<div class="mt-0.5 text-xs text-gcp-muted">
								Created {formatDate(admin.created_at)}
								{#if admin.last_login}— Last login {formatDate(admin.last_login)}{/if}
							</div>
						</div>
						{#if admin.username !== currentUser?.username}
							<button onclick={() => askDelete(admin.username)}
								class="rounded px-2 py-1 text-xs text-gcp-red hover:bg-red-50 transition-colors">
								Remove
							</button>
						{/if}
					</div>
				{/each}
			</div>
		{/if}

		<h3 class="mb-3 text-xs font-semibold uppercase tracking-wide text-gcp-muted">Add Administrator</h3>
		<form onsubmit={addAdmin} class="flex flex-wrap gap-2">
			<input bind:value={newAdminUser} placeholder="Username" required
				class="input-field w-36" autocomplete="username" />
			<input type="password" bind:value={newAdminPw} placeholder="Password (min 8)" required minlength="8"
				class="input-field w-44" autocomplete="new-password" />
			<button type="submit" disabled={addLoading}
				class="rounded bg-gcp-green px-3 py-1.5 text-xs text-white hover:opacity-90 disabled:opacity-60 transition-colors">
				<Plus size={12} class="inline mr-1" />{addLoading ? 'Adding…' : 'Add Admin'}
			</button>
		</form>
	</div>
</div>
