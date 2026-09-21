<script lang="ts">
  import TrendingUpIcon from "@lucide/svelte/icons/trending-up";
  import TrendingDownIcon from "@lucide/svelte/icons/trending-down";
  import ScaleIcon from "@lucide/svelte/icons/scale";
  import * as Card from "$lib/components/ui/card";
  import { Badge } from "$lib/components/ui/badge";
  import { varianceState } from "./money";
  import { t } from "./i18n";
  import AnimatedNumber from "./AnimatedNumber.svelte";

  let { variance }: { variance: number } = $props();
  const state = $derived(varianceState(variance));
  const label = $derived(
    state === "over" ? $t.varianceOver : state === "short" ? $t.varianceShort : $t.varianceBalanced
  );
</script>

<Card.Root class="overflow-hidden">
  <Card.Header class="pb-2">
    <Card.Title class="text-sm font-medium">{$t.variance}</Card.Title>
    <Card.Description>{$t.varianceFormula}</Card.Description>
  </Card.Header>
  <Card.Content class="flex items-center justify-between gap-3">
    <span class="min-w-0 text-3xl leading-tight font-extrabold tracking-tight break-words sm:text-4xl">
      <AnimatedNumber value={variance} signed announce />
    </span>
    {#if state === "over"}
      <Badge class="bg-success text-success-foreground hover:bg-success/90 shrink-0 gap-1">
        <TrendingUpIcon aria-hidden="true" />{label}
      </Badge>
    {:else if state === "short"}
      <Badge variant="destructive" class="shrink-0 gap-1">
        <TrendingDownIcon aria-hidden="true" />{label}
      </Badge>
    {:else}
      <Badge variant="secondary" class="shrink-0 gap-1">
        <ScaleIcon aria-hidden="true" />{label}
      </Badge>
    {/if}
  </Card.Content>
</Card.Root>
