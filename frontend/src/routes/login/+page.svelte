<script lang="ts">
	import { goto } from '$app/navigation';
	import { authApi } from '$lib/api/auth';
	import { setAuth } from '$lib/stores/auth.svelte';

	let username    = $state('');
	let password    = $state('');
	let loading     = $state(false);
	let keepSigned  = $state(false);
	let error       = $state('');

	async function handleLogin(e: Event) {
		e.preventDefault();
		loading = true;
		error = '';
		try {
			await authApi.login(username, password);
			const me = await authApi.me();
			setAuth(me);
			goto('/dashboard');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Login failed';
		} finally {
			loading = false;
		}
	}
</script>

<div class="login-root">

	<!-- ═══ LEFT PANEL ═══ -->
	<div class="left-panel">

		<!-- Grid overlay -->
		<div class="grid-overlay"></div>

		<!-- Geometric accent blocks — fractured-glass stack -->
		<div class="geo geo-tr-1"></div>
		<div class="geo geo-tr-2"></div>
		<div class="geo geo-tr-3"></div>
		<div class="geo geo-bl-1"></div>
		<div class="geo geo-bl-2"></div>

		<!-- Scattered small squares -->
		<div class="dot" style="width:22px;height:22px;top:44%;left:18%;opacity:.18;"></div>
		<div class="dot" style="width:10px;height:10px;top:48%;left:26%;opacity:.28;"></div>
		<div class="dot" style="width:16px;height:16px;top:36%;left:72%;opacity:.14;"></div>
		<div class="dot" style="width: 8px;height: 8px;top:58%;left:64%;opacity:.22;"></div>
		<div class="dot" style="width:14px;height:14px;top:72%;left:40%;opacity:.16;"></div>
		<div class="dot" style="width:24px;height:24px;top:22%;left:56%;opacity:.10;"></div>

		<!-- Logo -->
		<div class="left-top">
			<div class="logo-box">
				<div class="logo-inner"></div>
			</div>
			<span class="company-name">IT KMITL</span>
		</div>

		<!-- Hero text -->
		<div class="left-body">
			<p class="subtitle">Nice to see you again</p>
			<h1 class="hero-heading">WELCOME<br>BACK</h1>
			<div class="underline-accent"></div>
			<p class="hero-body">
				Manage Samba file shares, Active Directory users,
				and server permissions from one secure control panel.
				Authorised administrators only.
			</p>
		</div>
	</div>

	<!-- ═══ RIGHT PANEL ═══ -->
	<div class="right-panel">
		<div class="form-wrap">
			<h2 class="form-heading">Login Account</h2>
			<p class="form-sub">Enter your administrator credentials to access the SMB management panel.</p>

			<form onsubmit={handleLogin} class="form">

				<div class="field">
					<label for="username" class="field-label">Username</label>
					<input
						id="username"
						type="text"
						bind:value={username}
						autocomplete="username"
						required
						placeholder="admin"
						class="field-input"
					/>
				</div>

				<div class="field">
					<label for="password" class="field-label">Password</label>
					<input
						id="password"
						type="password"
						bind:value={password}
						autocomplete="current-password"
						required
						placeholder="••••••••"
						class="field-input"
					/>
				</div>

				<div class="form-row">
					<label class="keep-label">
						<input type="checkbox" bind:checked={keepSigned} class="keep-check" />
						Keep me signed in
					</label>
				</div>

				{#if error}
					<div class="error-box">{error}</div>
				{/if}

				<button type="submit" disabled={loading} class="submit-btn" class:loading>
					{loading ? 'SIGNING IN…' : 'SIGN IN'}
				</button>
			</form>
		</div>
	</div>
</div>

<style>
	/* ── Root ── */
	.login-root {
		display: flex;
		min-height: 100vh;
		font-family: 'Inter', system-ui, -apple-system, sans-serif;
	}

	/* ── Left panel ── */
	.left-panel {
		position: relative;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		width: 50%;
		overflow: hidden;
		background: linear-gradient(135deg, #5BA4F5 0%, #1565C0 55%, #0D3A8A 100%);
	}

	/* Grid overlay */
	.grid-overlay {
		position: absolute;
		inset: 0;
		background-image:
			linear-gradient(rgba(255,255,255,.07) 1px, transparent 1px),
			linear-gradient(90deg, rgba(255,255,255,.07) 1px, transparent 1px);
		background-size: 42px 42px;
		pointer-events: none;
	}

	/* Geometric accent blocks */
	.geo {
		position: absolute;
		pointer-events: none;
	}
	/* top-right stack */
	.geo-tr-1 {
		width: 340px; height: 340px;
		top: -90px; right: -90px;
		background: rgba(255,255,255,.09);
		transform: rotate(14deg);
	}
	.geo-tr-2 {
		width: 220px; height: 220px;
		top: -20px; right: 30px;
		background: rgba(255,255,255,.07);
		transform: rotate(14deg);
	}
	.geo-tr-3 {
		width: 120px; height: 120px;
		top: 80px; right: 110px;
		background: rgba(255,255,255,.12);
		transform: rotate(14deg);
	}
	/* bottom-left stack */
	.geo-bl-1 {
		width: 280px; height: 280px;
		bottom: -70px; left: -70px;
		background: rgba(255,255,255,.08);
		transform: rotate(-18deg);
	}
	.geo-bl-2 {
		width: 160px; height: 160px;
		bottom: 40px; left: 40px;
		background: rgba(255,255,255,.06);
		transform: rotate(-18deg);
	}

	/* Scattered accent squares */
	.dot {
		position: absolute;
		background: white;
		pointer-events: none;
	}

	/* Logo row */
	.left-top {
		position: relative;
		z-index: 10;
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 36px 44px;
	}
	.logo-box {
		width: 38px; height: 38px;
		background: rgba(255,255,255,.2);
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.logo-inner {
		width: 18px; height: 18px;
		background: white;
	}
	.company-name {
		color: rgba(255,255,255,.92);
		font-size: 11px;
		font-weight: 700;
		letter-spacing: .14em;
		text-transform: uppercase;
	}

	/* Hero text */
	.left-body {
		position: relative;
		z-index: 10;
		padding: 0 44px 56px;
	}
	.subtitle {
		color: rgba(255,255,255,.7);
		font-size: 12px;
		font-weight: 500;
		letter-spacing: .1em;
		text-transform: uppercase;
		margin: 0 0 14px;
	}
	.hero-heading {
		color: white;
		font-size: 46px;
		font-weight: 900;
		line-height: .95;
		letter-spacing: -.02em;
		margin: 0 0 18px;
	}
	.underline-accent {
		width: 44px;
		height: 3px;
		background: rgba(255,255,255,.7);
		margin-bottom: 20px;
	}
	.hero-body {
		color: rgba(255,255,255,.58);
		font-size: 13px;
		line-height: 1.75;
		max-width: 290px;
		margin: 0;
	}

	/* ── Right panel ── */
	.right-panel {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		background: #ffffff;
		padding: 48px 32px;
	}
	.form-wrap {
		width: 100%;
		max-width: 340px;
	}

	.form-heading {
		font-size: 26px;
		font-weight: 800;
		color: #1565C0;
		margin: 0 0 10px;
		letter-spacing: -.01em;
	}
	.form-sub {
		font-size: 13px;
		color: #5f6368;
		line-height: 1.65;
		margin: 0 0 32px;
	}

	/* Form */
	.form {
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.field-label {
		font-size: 11px;
		font-weight: 700;
		color: #202124;
		letter-spacing: .08em;
		text-transform: uppercase;
	}
	.field-input {
		width: 100%;
		padding: 11px 13px;
		border-radius: 0;
		border: 1px solid #dde1e7;
		border-left: 3px solid #1a73e8;
		background: #f5f7fa;
		font-size: 14px;
		color: #202124;
		outline: none;
		box-sizing: border-box;
		transition: border-color .15s, background .15s;
	}
	.field-input::placeholder {
		color: #9aa0a6;
	}
	.field-input:focus {
		border-color: #1a73e8;
		background: #ffffff;
		box-shadow: 0 0 0 3px rgba(26,115,232,.12);
	}

	/* Checkbox row */
	.form-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.keep-label {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 13px;
		color: #5f6368;
		cursor: pointer;
		user-select: none;
	}
	.keep-check {
		width: 15px;
		height: 15px;
		border-radius: 0;
		accent-color: #1a73e8;
		cursor: pointer;
		flex-shrink: 0;
	}

	/* Error */
	.error-box {
		background: #fce8e6;
		border-left: 3px solid #c5221f;
		padding: 10px 13px;
		font-size: 13px;
		color: #c5221f;
		line-height: 1.5;
	}

	/* Submit button */
	.submit-btn {
		width: 100%;
		padding: 14px;
		border-radius: 0;
		border: none;
		background: #1a73e8;
		color: white;
		font-size: 13px;
		font-weight: 800;
		letter-spacing: .1em;
		text-transform: uppercase;
		cursor: pointer;
		transition: background .15s;
		box-shadow: 0 2px 12px rgba(26,115,232,.35);
	}
	.submit-btn:hover:not(:disabled) {
		background: #1557b0;
	}
	.submit-btn:disabled,
	.submit-btn.loading {
		opacity: .65;
		cursor: not-allowed;
	}

	/* Responsive: stack on small screens */
	@media (max-width: 768px) {
		.login-root { flex-direction: column; }
		.left-panel { width: 100%; min-height: 220px; }
		.left-body  { padding-bottom: 36px; }
		.hero-heading { font-size: 32px; }
	}
</style>
