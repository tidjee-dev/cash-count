<script lang="ts">
  import { push } from "svelte-spa-router";
  import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
  import { buttonVariants } from "$lib/components/ui/button";
  import { t } from "$lib/i18n";

  function back() {
    push("/history");
  }
</script>

<div class="mx-auto flex w-full max-w-5xl flex-1 flex-col items-start gap-4 px-4 py-6 pb-16">
  <h1 class="text-xl font-bold tracking-tight">{$t.notFoundTitle}</h1>
  <p class="text-muted-foreground text-sm">{$t.notFoundBody}</p>
  <button class={buttonVariants({ variant: "outline", size: "sm" })} onclick={back}>
    <ArrowLeftIcon data-icon="inline-start" aria-hidden="true" />{$t.backToHistory}
  </button>
</div>
