<script lang="ts">
	import * as api from "../api";
	import { loginMode } from "../config.svelte";
	import { forgetAll } from "../data.svelte";
	import Icon from "../components/Icon.svelte";

	// gotrue logs in by email, db mode by the PostgreSQL role
	const userLabel = loginMode === "gotrue" ? "Email" : "Role";

	let username = $state("");
	let password = $state("");
	let showPassword = $state(false);
	let isLoading = $state(false);
	let errorMessage = $state("");

	// After a successful login App leaves this page.
	async function handleLogin(event: Event) {
		event.preventDefault();
		isLoading = true;
		errorMessage = "";
		const result = await api.login(username, password);
		isLoading = false;
		// another role can see other tables
		if (result.success) forgetAll();
		if (!result.success) {
			errorMessage = result.error || "Invalid username or password";
		}
	}
</script>

<div class="login">
	<form class="card" onsubmit={handleLogin}>
		<div class="brand">
			<span class="mark"><Icon name="database-2" size={16} /></span>
			<h1>Sign in to smoothdb</h1>
		</div>

		<div class="field">
			<label class="field-label" for="username">{userLabel}</label>
			<!-- svelte-ignore a11y_autofocus -->
			<input
				id="username"
				class="input"
				type="text"
				bind:value={username}
				placeholder={loginMode === "gotrue" ? "name@company.com" : "postgres"}
				autocomplete="username"
				autofocus
				required
				disabled={isLoading}
			/>
		</div>

		<div class="field">
			<label class="field-label" for="password">Password</label>
			<div class="password">
				<input
					id="password"
					class="input"
					type={showPassword ? "text" : "password"}
					bind:value={password}
					autocomplete="current-password"
					required
					disabled={isLoading}
				/>
				<button
					type="button"
					class="btn btn-ghost btn-sm btn-icon reveal"
					onclick={() => (showPassword = !showPassword)}
					disabled={isLoading}
					aria-label={showPassword ? "Hide password" : "Show password"}
				>
					<Icon name="eye" size={14} />
				</button>
			</div>
		</div>

		{#if errorMessage}
			<div class="banner banner-danger" role="alert">
				<Icon name="error-warning" size={15} />
				{errorMessage}
			</div>
		{/if}

		<button type="submit" class="btn btn-primary submit" disabled={isLoading}>
			{isLoading ? "Signing in..." : "Sign in"}
		</button>
	</form>
</div>

<style>
	.login {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 100vh;
		padding: 16px;
		background: var(--bg-subtle);
	}
	.card {
		display: flex;
		flex-direction: column;
		gap: 16px;
		width: 100%;
		max-width: 340px;
		padding: 24px;
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		background: var(--bg);
		box-shadow: var(--shadow-pop);
	}
	.brand {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 12px;
		margin-bottom: 4px;
	}
	.mark {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: 7px;
		background: var(--accent);
		color: var(--accent-fg);
	}
	h1 {
		font-size: 16px;
		font-weight: 600;
	}
	.password {
		position: relative;
	}
	.password .input {
		padding-right: 32px;
	}
	.reveal {
		position: absolute;
		top: 2px;
		right: 2px;
	}
	.submit {
		height: 32px;
		margin-top: 4px;
	}
</style>
