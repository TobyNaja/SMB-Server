<script lang="ts">
	import './layout.css';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import favicon from '$lib/assets/favicon.svg';
	import Toast from '$lib/components/Toast.svelte';
	import { authApi } from '$lib/api/auth';
	import { getUser, setAuth, clearAuth, isAuthenticated } from '$lib/stores/auth.svelte';
	import {
		Server, LayoutDashboard, Folder, User, Users, Building2,
		Shield, ClipboardList, Settings, LogOut, Menu, X
	} from 'lucide-svelte';

	let { children } = $props();

	const navItems = [
		{ href: '/dashboard', label: 'Dashboard',        icon: LayoutDashboard },
		{ href: '/shares',    label: 'Shares',            icon: Folder          },
		{ href: '/users',     label: 'Users',             icon: User            },
		{ href: '/groups',    label: 'Groups',            icon: Users           },
		{ href: '/ad',        label: 'Active Directory',  icon: Building2       },
		{ href: '/builtin',   label: 'Builtin Groups',    icon: Shield          },
		{ href: '/audit',     label: 'Audit Log',         icon: ClipboardList   },
		{ href: '/settings',  label: 'Settings',          icon: Settings        }
	];

	const isLoginPage  = $derived($page.url.pathname === '/login');
	const isSetupPage  = $derived($page.url.pathname === '/setup');
	const isPublicPage = $derived(isLoginPage || isSetupPage);

	let sidebarOpen = $state(true);

	// Session timeout: JWT is 24h — warn 10 min before expiry
	let sessionWarning = $state(false);
	let sessionTimer: ReturnType<typeof setTimeout> | null = null;

	function scheduleSessionWarning() {
		const token = localStorage.getItem('access_token');
		if (!token) return;
		try {
			const payload = JSON.parse(atob(token.split('.')[1]));
			const expiresAt = payload.exp * 1000;
			const warnAt = expiresAt - 10 * 60 * 1000;
			const delay = warnAt - Date.now();
			if (delay > 0) {
				sessionTimer = setTimeout(() => { sessionWarning = true; }, delay);
			}
		} catch { /* malformed token — ignore */ }
	}

	onMount(async () => {
		try {
			const status = await fetch('/auth/status').then(r => r.json());
			if (status.setup_required && !isSetupPage) { goto('/setup'); return; }
			if (!status.setup_required && isSetupPage) { goto('/login'); return; }
		} catch { /* network error — let the page handle it */ }

		if (isPublicPage) return;
		if (!isAuthenticated()) { goto('/login'); return; }
		try {
			const me = await authApi.me();
			setAuth(me, localStorage.getItem('access_token') ?? '');
			scheduleSessionWarning();
		} catch {
			clearAuth();
			goto('/login');
		}

		return () => { if (sessionTimer) clearTimeout(sessionTimer); };
	});

	async function handleLogout() {
		try { await authApi.logout(); } catch { /* ignore */ }
		finally { clearAuth(); goto('/login'); }
	}

	function isActive(href: string) {
		return $page.url.pathname === href || $page.url.pathname.startsWith(href + '/');
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>SMB Manager</title>
</svelte:head>

<Toast />

{#if isPublicPage}
	{@render children()}
{:else}
	<div class="flex h-screen overflow-hidden">
		<!-- Sidebar -->
		<aside
			class="flex flex-col border-r border-gcp-border bg-white transition-all duration-200
				{sidebarOpen ? 'w-56' : 'w-14'} flex-none"
		>
			<!-- Sidebar header -->
			<div class="flex items-center gap-2 bg-gcp-nav px-3 py-3.5 text-white">
				<Server size={20} class="flex-none" />
				{#if sidebarOpen}
					<span class="truncate text-sm font-semibold tracking-wide">SMB Manager</span>
				{/if}
			</div>

			<!-- Nav items -->
			<nav class="flex-1 overflow-y-auto py-1">
				{#each navItems as item}
					{@const Icon = item.icon}
					{@const active = isActive(item.href)}
					<a
						href={item.href}
						title={!sidebarOpen ? item.label : undefined}
						class="relative flex items-center gap-3 px-3 py-2.5 text-sm transition-colors
							{active
								? 'bg-gcp-blue-light text-gcp-blue font-medium'
								: 'text-gcp-muted hover:bg-gcp-bg hover:text-gcp-dark'}"
					>
						{#if active}
							<span class="absolute left-0 top-0 h-full w-0.5 rounded-r bg-gcp-blue"></span>
						{/if}
						<Icon size={16} class="flex-none" />
						{#if sidebarOpen}
							<span>{item.label}</span>
						{/if}
					</a>
				{/each}
			</nav>

			<!-- User + logout -->
			<div class="border-t border-gcp-border p-3">
				{#if sidebarOpen && getUser()}
					<div class="mb-2 truncate text-xs text-gcp-muted">
						<span class="font-medium text-gcp-dark">{getUser()?.username}</span>
					</div>
				{/if}
				<button
					onclick={handleLogout}
					title="Logout"
					class="flex w-full items-center gap-2 rounded px-2 py-1.5 text-sm text-gcp-muted
						hover:bg-red-50 hover:text-gcp-red transition-colors"
				>
					<LogOut size={14} class="flex-none" />
					{#if sidebarOpen}<span>Logout</span>{/if}
				</button>
			</div>
		</aside>

		<!-- Right column: topbar + content -->
		<div class="flex flex-1 flex-col overflow-hidden">
			<!-- Topbar -->
			<header class="flex h-12 flex-none items-center gap-3 border-b border-gcp-border bg-white px-4">
				<button
					onclick={() => (sidebarOpen = !sidebarOpen)}
					class="rounded p-1.5 text-gcp-muted hover:bg-gcp-bg transition-colors"
					aria-label="Toggle sidebar"
				>
					{#if sidebarOpen}
						<X size={18} />
					{:else}
						<Menu size={18} />
					{/if}
				</button>

				<span class="text-sm font-medium text-gcp-dark">
					{navItems.find(n => isActive(n.href))?.label ?? 'SMB Manager'}
				</span>

				{#if getUser()}
					<div class="ml-auto flex items-center gap-2 rounded border border-gcp-border px-2.5 py-1 text-xs text-gcp-dark">
						<User size={12} />
						{getUser()?.username}
					</div>
				{/if}
			</header>

			<!-- Session expiry warning -->
			{#if sessionWarning}
				<div class="flex items-center gap-3 border-b border-yellow-200 bg-yellow-50 px-4 py-2 text-xs text-yellow-800">
					<span class="font-medium">Session expiring soon.</span>
					<span>Log out and log in again to stay active.</span>
					<button onclick={() => (sessionWarning = false)} class="ml-auto text-yellow-600 hover:text-yellow-800">
						<X size={14} />
					</button>
				</div>
			{/if}

			<!-- Page content -->
			<main class="flex-1 overflow-auto bg-gcp-bg">
				<div class="p-6">
					{@render children()}
				</div>
			</main>
		</div>
	</div>
{/if}
