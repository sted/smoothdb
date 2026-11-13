<script lang="ts">

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
	$effect(() => {
		const input = data;
		if (input instanceof Promise) {
			input.then((d) => {
				items = normalizeData(d);
			});
		} else {
			items = normalizeData(input);
		}
	});
</script>

<select {id} class="input" bind:value>
	{#each items as item}
		<option value={item.value}>{item.key}</option>
	{/each}
</select>
