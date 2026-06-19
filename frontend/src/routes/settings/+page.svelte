<script lang="ts">
	import { onMount } from 'svelte';
	import { authApi, type AdminInfo } from '$lib/api/auth';
	import { getUser } from '$lib/stores/auth.svelte';
	import { toast, toastError } from '$lib/stores/toast.svelte';

	let admins = $state<AdminInfo[]>([]);
	let loading = $state(true);
	let currentUser = $derived(getUser());

	// Change password form
	let oldPw = $state('');
	let newPw = $state('');
	let confirmPw = $state('');
	let pwLoading = $state(false);

	// Add admin form
	let newAdminUser = $state('');
	let newAdminPw = $state('');
	let addLoading = $state(false);

	async function loadAdmins() {
		loading = true;
		try {
			const r = await authApi.listAdmins();
			admins = r.admins;
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

	async function deleteAdmin(username: string) {
		if (!confirm(`Remove admin '${username}'?`)) return;
		try {
			await authApi.deleteAdmin(username);
			toast(`Admin '${username}' removed`);
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

<div class="max-w-2xl space-y-8">
	<h1 class="text-lg font-semibold text-gray-800 dark:text-white">Settings</h1>

	<!-- Change password -->
	<section class="rounded-xl bg-white p-6 shadow-sm dark:bg-gray-800">
		<h2 class="mb-4 font-medium text-gray-800 dark:text-white">Change Your Password</h2>
		<form onsubmit={changePassword} class="space-y-3">
			<div>
				<label for="old-pw" class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
					Current password
				</label>
				<input id="old-pw" type="password" bind:value={oldPw} required
					class="input-field w-full" placeholder="••••••••" autocomplete="current-password" />
			</div>
			<div>
				<label for="new-pw" class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
					New password (min 8 characters)
				</label>
				<input id="new-pw" type="password" bind:value={newPw} required minlength="8"
					class="input-field w-full" placeholder="••••••••" autocomplete="new-password" />
			</div>
			<div>
				<label for="confirm-pw" class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
					Confirm new password
				</label>
				<input id="confirm-pw" type="password" bind:value={confirmPw} required
					class="input-field w-full" placeholder="••••••••" autocomplete="new-password" />
			</div>
			<button type="submit" disabled={pwLoading}
				class="rounded bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700 disabled:opacity-60">
				{pwLoading ? 'Updating…' : 'Update Password'}
			</button>
		</form>
	</section>

	<!-- Admin management -->
	<section class="rounded-xl bg-white p-6 shadow-sm dark:bg-gray-800">
		<h2 class="mb-4 font-medium text-gray-800 dark:text-white">Administrators</h2>

		{#if loading}
			<div class="mb-4 text-sm text-gray-400">Loading…</div>
		{:else}
			<div class="mb-5 divide-y divide-gray-50 rounded-lg border border-gray-100 dark:divide-gray-700 dark:border-gray-700">
				{#each admins as admin}
					<div class="flex items-center justify-between px-4 py-3">
						<div>
							<div class="flex items-center gap-2 text-sm font-medium text-gray-800 dark:text-white">
								🔑 {admin.username}
								{#if admin.username === currentUser?.username}
									<span class="rounded-full bg-blue-100 px-2 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">you</span>
								{/if}
							</div>
							<div class="mt-0.5 text-xs text-gray-400">
								Created {formatDate(admin.created_at)}
								{#if admin.last_login}— Last login {formatDate(admin.last_login)}{/if}
							</div>
						</div>
						{#if admin.username !== currentUser?.username}
							<button onclick={() => deleteAdmin(admin.username)}
								class="rounded bg-red-600 px-2 py-1 text-xs text-white hover:bg-red-700">
								Remove
							</button>
						{/if}
					</div>
				{/each}
			</div>
		{/if}

		<h3 class="mb-3 text-sm font-medium text-gray-700 dark:text-gray-300">Add Administrator</h3>
		<form onsubmit={addAdmin} class="flex flex-wrap gap-3">
			<input bind:value={newAdminUser} placeholder="Username" required
				class="input-field w-40" autocomplete="username" />
			<input type="password" bind:value={newAdminPw} placeholder="Password (min 8)" required minlength="8"
				class="input-field w-44" autocomplete="new-password" />
			<button type="submit" disabled={addLoading}
				class="rounded bg-green-600 px-4 py-2 text-sm text-white hover:bg-green-700 disabled:opacity-60">
				{addLoading ? 'Adding…' : 'Add Admin'}
			</button>
		</form>
	</section>
</div>

