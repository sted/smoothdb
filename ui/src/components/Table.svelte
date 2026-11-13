<script lang="ts">
	import { router } from "../routes";
	import { makeRequest } from "../api";
	import type { Data } from "../api";
	import Grid from "./Grid.svelte";

	interface Props {
		dataUrl: string;
		rowClick: (d: Data) => void;
		rowEdit: (d: Data) => void;
	}
	let { dataUrl, rowClick, rowEdit }: Props = $props();

	let data: Data[] = $state([]);
	let loading = $state(false);

	$effect(() => {
		dataUrl;
		refresh();
	});

	export async function refresh() {
		if (!dataUrl) return;
		loading = true;
		const headers: Record<string, string> = router.schema ? { "Accept-Profile": router.schema } : {};
		const response = await makeRequest<Data[]>(dataUrl, { headers });
		loading = false;
		data = response.success && Array.isArray(response.data) ? response.data : [];
	}
</script>

<Grid rows={data} {loading} onEdit={rowEdit} onRowClick={rowClick} />
