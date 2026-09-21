<script lang="ts">
	import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
	import { cn } from "$lib/utils.js";

	let { class: className }: { class?: string } = $props();
</script>

<LoaderCircleIcon data-slot="spinner" data-icon="inline-start" class={cn("size-4 animate-spin", className)} />
