<script lang="ts">
  import SunriseIcon from "@lucide/svelte/icons/sunrise";
  import SunsetIcon from "@lucide/svelte/icons/sunset";
  import ClipboardCheckIcon from "@lucide/svelte/icons/clipboard-check";
  import HandCoinsIcon from "@lucide/svelte/icons/hand-coins";
  import CircleDotIcon from "@lucide/svelte/icons/circle-dot";
  import { Badge } from "$lib/components/ui/badge";
  import { countTypeLabel } from "./format";
  import { t } from "./i18n";

  let { type }: { type: string } = $props();

  const Icon = $derived(
    type === "shift_open"
      ? SunriseIcon
      : type === "shift_close"
        ? SunsetIcon
        : type === "audit"
          ? ClipboardCheckIcon
          : type === "donation_urne"
            ? HandCoinsIcon
            : CircleDotIcon
  );
</script>

<Badge variant="secondary" class="gap-1">
  <Icon aria-hidden="true" />{countTypeLabel(type, $t)}
</Badge>
