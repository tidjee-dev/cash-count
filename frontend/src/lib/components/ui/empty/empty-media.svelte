<script lang="ts" module>
	import { type VariantProps, tv } from "tailwind-variants";

	const emptyMediaVariants = tv({
		base: "mb-2 flex shrink-0 items-center justify-center [&_svg]:pointer-events-none [&_svg]:shrink-0",
		variants: {
			variant: {
				default: "bg-transparent",
				icon: "bg-muted text-foreground flex size-10 shrink-0 items-center justify-center rounded-lg [&_svg:not([class*='size-'])]:size-6",
			},
		},
		defaultVariants: { variant: "default" },
	});

	export type EmptyMediaVariant = VariantProps<typeof emptyMediaVariants>["variant"];
</script>

<script lang="ts">
	import { cn, type WithElementRef } from "$lib/utils.js";
	import type { HTMLAttributes } from "svelte/elements";

	let {
		ref = $bindable(null),
		class: className,
		variant = "default",
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> & { variant?: EmptyMediaVariant } = $props();
</script>

<div
	bind:this={ref}
	data-slot="empty-media"
	data-variant={variant}
	class={cn(emptyMediaVariants({ variant }), className)}
	{...restProps}
>
	{@render children?.()}
</div>
