<script lang="ts">
  import MinusIcon from "@lucide/svelte/icons/minus";
  import PlusIcon from "@lucide/svelte/icons/plus";
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import { formatCents } from "./money";
  import { currencySymbol } from "./ui";
  import { t, msg, locale } from "./i18n";
  import AnimatedNumber from "./AnimatedNumber.svelte";
  import type { DenomState } from "./types";

  let {
    denom,
    qty,
    onChange,
    autofocus = false,
  }: {
    denom: DenomState;
    qty: number;
    onChange: (q: number) => void;
    autofocus?: boolean;
  } = $props();

  const MAX = 99999;
  let inputEl: HTMLInputElement | null = $state(null);
  let focused = $state(false);
  // Seeded from the initial qty; kept in sync by the effect below.
  // svelte-ignore state_referenced_locally
  let lastEmitted = $state(qty);
  // Empty field while idle means zero — cleaner than a forced "0".
  // svelte-ignore state_referenced_locally
  let text = $state(qty === 0 ? "" : String(qty));

  $effect(() => {
    if (!focused && qty !== lastEmitted) {
      text = qty === 0 ? "" : String(qty);
      lastEmitted = qty;
    }
  });

  $effect(() => {
    if (autofocus) inputEl?.focus();
  });

  function commit(raw: string) {
    const digits = raw.replace(/[^0-9]/g, "").replace(/^0+(?=\d)/, "");
    if (digits === "") {
      text = "";
      lastEmitted = 0;
      onChange(0);
      return;
    }
    const q = Math.min(MAX, parseInt(digits, 10));
    text = String(q);
    lastEmitted = q;
    onChange(q);
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === "ArrowUp") {
      e.preventDefault();
      onChange(Math.min(MAX, qty + 1));
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      onChange(Math.max(0, qty - 1));
    } else if (e.key === "Enter") {
      e.preventDefault();
      (e.target as HTMLElement).blur();
      focusNext();
    } else if (e.key === "Escape") {
      e.preventDefault();
      text = qty === 0 ? "" : String(qty);
      (e.target as HTMLInputElement).blur();
    }
  }

  /** Move focus to the next denomination quantity input for rapid entry. */
  function focusNext() {
    const inputs = Array.from(
      document.querySelectorAll<HTMLInputElement>("input[data-denom-qty]"),
    );
    const i = inputs.indexOf(inputEl as HTMLInputElement);
    if (i >= 0 && i + 1 < inputs.length) {
      inputs[i + 1].focus();
      return;
    }
    // Last row: continue into the details card. The expected control varies
    // (select trigger, custom input, or absent in urne mode), so try each.
    for (const id of ["expected", "expected-custom", "note"]) {
      const el = document.getElementById(id) as HTMLElement | null;
      if (el) {
        el.focus();
        return;
      }
    }
  }
</script>

<div class="grid grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-2 sm:gap-3 py-2">
  <div class="min-w-0">
    <div class="text-sm font-medium">{denom.label}</div>
    <div class="text-muted-foreground text-xs tabular-nums">
      {msg($t.each, {
        amount: formatCents(denom.value_cents, $currencySymbol, $locale),
      })}
    </div>
  </div>
  <div class="flex items-center gap-1.5">
    <Button
      variant="outline"
      size="icon-lg"
      aria-label={msg($t.decreaseOf, { label: denom.label })}
      aria-disabled={qty <= 0}
      onclick={() => qty > 0 && onChange(qty - 1)}
    >
      <MinusIcon aria-hidden="true" />
    </Button>
    <Input
      type="text"
      inputmode="numeric"
      pattern="[0-9]*"
      placeholder="0"
      data-denom-qty={denom.id}
      max={MAX}
      min="0"
      aria-label={msg($t.qtyOf, { label: denom.label })}
      value={text}
      bind:ref={inputEl}
      oninput={(e) => commit((e.target as HTMLInputElement).value)}
      onfocus={(e) => {
        focused = true;
        (e.target as HTMLInputElement).select();
      }}
      onblur={() => {
        focused = false;
        text = qty === 0 ? "" : String(qty);
        lastEmitted = qty;
      }}
      onkeydown={onKey}
      class="w-20 text-center tabular-nums font-bold text-xl"
    />
    <Button
      variant="outline"
      size="icon-lg"
      aria-label={msg($t.increaseOf, { label: denom.label })}
      onclick={() => onChange(Math.min(MAX, qty + 1))}
    >
      <PlusIcon aria-hidden="true" />
    </Button>
  </div>
  <div class="min-w-16 text-right text-sm font-semibold sm:min-w-21">
    <AnimatedNumber value={qty * denom.value_cents} />
  </div>
</div>
