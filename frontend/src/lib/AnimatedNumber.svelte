<script lang="ts">
  import { tweened } from "svelte/motion";
  import { cubicOut } from "svelte/easing";
  import { formatCents, formatSignedCents } from "./money";
  import { currencySymbol } from "./ui";
  import { locale } from "./i18n";

  let {
    value,
    signed = false,
    announce = false,
    class: className = "",
  }: { value: number; signed?: boolean; announce?: boolean; class?: string } = $props();

  const reduced =
    typeof matchMedia !== "undefined" &&
    matchMedia("(prefers-reduced-motion: reduce)").matches;
  // Intentionally seeded with the initial value so the first render is exact.
  // svelte-ignore state_referenced_locally
  const display = tweened(value, { duration: reduced ? 0 : 250, easing: cubicOut });

  $effect(() => {
    display.set(value);
  });

  const text = $derived(
    signed
      ? formatSignedCents(Math.round($display), $currencySymbol, $locale)
      : formatCents(Math.round($display), $currencySymbol, $locale)
  );
</script>

<span class="tabular-nums {className}" role={announce ? "status" : undefined}>{text}</span>
