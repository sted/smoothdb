<script lang="ts">
	import RiExpandUpDownLine from "svelte-remixicon/RiExpandUpDownLine.svelte";

	type InputData = Promise<any[]> | Promise<object> | any[] | object;
	type NormalizedData = { key: string; value: string }[];

	interface Props {
		id: string;
		value: string;
		data: InputData;
		fieldName?: string;
		fieldValue?: string;
	}

	let { id, value = $bindable(), data, fieldName = "", fieldValue = "" }: Props = $props();

	function normalizeData(input: InputData): NormalizedData {
		if (fieldValue === "") {
			fieldValue = fieldName;
		}
		if (Array.isArray(input)) {
			if (input.length > 0 && typeof input[0] === "object") {
				return input.map((item) => ({ key: item[fieldName], value: item[fieldValue] }));
			}
			return input.map((item) => ({ key: item, value: item }));
		} else if (typeof input === "object" && input !== null) {
			return Object.entries(input).map(([key, value]) => ({ key, value }));
		}
		return [];
	}
	let items: NormalizedData = $state([]);
	if (data instanceof Promise) {
		data.then((d) => {
			items = normalizeData(d);
		});
	} else {
		items = normalizeData(data);
	}
</script>

<div class="select-wrapper">
	<select {id} bind:value>
		{#each items as item}
			<option value={item.value}>{item.key}</option>
		{/each}
	</select>
	<div class="expand-widget">
		<RiExpandUpDownLine />
	</div>
</div>

<style>
	.select-wrapper {
		position: relative;
	}

	select {
		appearance: none;
	}

	.expand-widget {
		position: absolute;
		right: 0.5em;
		top: 50%;
		transform: translateY(-50%);
		pointer-events: none;
	}
</style>
