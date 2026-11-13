<script lang="ts">
	import Icon from "../components/Icon.svelte";
	interface Props {
		title?: string;
		message: string;
	}

	let { title = "Configuration error", message }: Props = $props();
</script>

<div class="page">
	<div class="card">
		<span class="icon-wrap"><Icon name="error-warning" size={18} /></span>
		<h1>{title}</h1>
		<p>{message}</p>
		<ul>
			<li>Check the server configuration file, <span class="mono">config.jsonc</span>.</li>
			<li>Restart the server after a change of the configuration.</li>
		</ul>
	</div>
</div>

<style>
	.page {
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
		gap: 10px;
		max-width: 480px;
		padding: 24px;
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		background: var(--bg);
	}
	.icon-wrap {
		display: inline-flex;
		width: 32px;
		height: 32px;
		align-items: center;
		justify-content: center;
		border-radius: 8px;
		background: var(--danger-soft);
		color: var(--danger);
	}
	h1 {
		font-size: 16px;
		font-weight: 600;
	}
	p {
		color: var(--text-2);
	}
	ul {
		padding-left: 18px;
		color: var(--text-3);
	}
</style>
