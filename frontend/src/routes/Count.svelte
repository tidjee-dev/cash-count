<script lang="ts">
  import { push } from "svelte-spa-router";
  import { toast } from "svelte-sonner";
  import PlusIcon from "@lucide/svelte/icons/plus";
  import * as Card from "$lib/components/ui/card";
  import * as Field from "$lib/components/ui/field";
  import * as Empty from "$lib/components/ui/empty";
  import * as Alert from "$lib/components/ui/alert";
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import { Textarea } from "$lib/components/ui/textarea";
  import { Separator } from "$lib/components/ui/separator";
  import { Skeleton } from "$lib/components/ui/skeleton";
  import { Spinner } from "$lib/components/ui/spinner";
  import * as RadioGroup from "$lib/components/ui/radio-group";
  import { Label } from "$lib/components/ui/label";
  import * as Select from "$lib/components/ui/select";
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import WalletIcon from "@lucide/svelte/icons/wallet";
  import SunriseIcon from "@lucide/svelte/icons/sunrise";
  import SunsetIcon from "@lucide/svelte/icons/sunset";
  import ClipboardCheckIcon from "@lucide/svelte/icons/clipboard-check";
  import HandCoinsIcon from "@lucide/svelte/icons/hand-coins";
  import { formatCents, formatSignedCents } from "../lib/money";
  import { fmtDate } from "../lib/format";
  import { currencySymbol } from "../lib/ui";
  import { t, msg, locale } from "../lib/i18n";
  import { loadAppData } from "../lib/backend";
  import { CountSvc } from "../lib/services";
  import DenomRow from "../lib/DenomRow.svelte";
  import VariancePanel from "../lib/VariancePanel.svelte";
  import AnimatedNumber from "../lib/AnimatedNumber.svelte";
  import CentsInput from "../lib/CentsInput.svelte";
  import { notifyDataChanged } from "../lib/state";
  import type { CountSummary, DenomState, DropDraft } from "../lib/types";

  let loading = $state(true);
  let loadError = $state("");
  let quantities = $state<Record<string, number>>({});
  let countType = $state("shift_open");
  let expected = $state(0);
  let expectedInvalid = $state(false);
  const expectedError = $derived(expectedInvalid ? $t.invalidAmount : "");
  let drops = $state<DropDraft[]>([]);
  let dropCents = $state(0);
  let dropInvalid = $state(false);
  let dropNote = $state("");
  let dropError = $state("");
  let note = $state("");
  let saving = $state(false);
  // Focus the first quantity only after a save resets the form (rapid entry
  // of the next count), never on initial load where it would steal focus.
  let firstInput = $state(false);
  let loadRetryBtn: HTMLElement | null = $state(null);
  let lastLoadError = $state("");

  // Locale-aware "0.00" placeholder for monetary inputs.
  const zeroPlaceholder = $derived(
    new Intl.NumberFormat($locale === "fr" ? "fr-FR" : "en-US", {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(0),
  );

  let denoms = $state<DenomState[]>([]);
  let counted = $state(0);
  let dropsTotal = $state(0);
  let variance = $state(0);
  let existingCounts = $state<CountSummary[]>([]);
  let expectedPick = $state("");
  const CUSTOM = "custom";

  /** Distinct counted totals from history, most recent first. */
  const countedOptions = $derived.by(() => {
    const seen = new Map<number, CountSummary>();
    for (const c of existingCounts) {
      if (!seen.has(c.counted_cents)) seen.set(c.counted_cents, c);
    }
    return [...seen.entries()].map(([amount, c]) => ({ amount, last: c }));
  });

  const typeOptions = $derived([
    {
      value: "shift_open",
      label: $t.typeOpen,
      hint: $t.typeOpenHint,
      icon: SunriseIcon,
    },
    {
      value: "shift_close",
      label: $t.typeClose,
      hint: $t.typeCloseHint,
      icon: SunsetIcon,
    },
    {
      value: "audit",
      label: $t.typeAudit,
      hint: $t.typeAuditHint,
      icon: ClipboardCheckIcon,
    },
    {
      value: "donation_urne",
      label: $t.typeDonationUrne,
      hint: $t.typeDonationUrneHint,
      icon: HandCoinsIcon,
    },
  ]);

  const isUrne = $derived(countType === "donation_urne");
  // Donation urns have no expected amount; the typed value is kept aside and
  // restored when switching back to another type.
  const effectiveExpected = $derived(isUrne ? 0 : expected);

  $effect(() => {
    load();
  });

  // Move focus to Retry only when a load failure freshly appears.
  $effect(() => {
    if (loadError && loadError !== lastLoadError) {
      lastLoadError = loadError;
      loadRetryBtn?.focus();
    } else if (!loadError) {
      lastLoadError = "";
    }
  });

  async function load() {
    loading = true;
    loadError = "";
    try {
      const data = await loadAppData();
      denoms = data.denoms.map((d) => ({ ...d, qty: quantities[d.id] ?? 0 }));
      quantities = Object.fromEntries(
        data.denoms.map((d) => [d.id, quantities[d.id] ?? 0]),
      );
      try {
        existingCounts = (await (CountSvc as any).ListCounts({
          limit: 50,
        })) as CountSummary[];
      } catch {
        existingCounts = [];
      }
    } catch (e) {
      loadError = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  async function refreshExisting() {
    try {
      existingCounts = (await (CountSvc as any).ListCounts({
        limit: 50,
      })) as CountSummary[];
    } catch {
      existingCounts = [];
    }
  }

  /** Expected amount comes from a past counted total (or a custom value). */
  function pickExpected(v: string) {
    if (v === CUSTOM || v === "") return;
    expected = Number(v);
    expectedInvalid = false;
  }

  $effect(() => {
    counted = denoms.reduce((s, d) => s + d.qty * d.value_cents, 0);
    dropsTotal = drops.reduce((s, d) => s + d.amount_cents, 0);
  });

  $effect(() => {
    variance = counted + dropsTotal - effectiveExpected;
  });

  function setQty(id: string, qty: number) {
    const q = Math.max(0, Math.min(999999, Math.trunc(qty) || 0));
    quantities = { ...quantities, [id]: q };
    denoms = denoms.map((d) => (d.id === id ? { ...d, qty: q } : d));
  }

  function addDrop() {
    dropError = "";
    if (dropInvalid || dropCents <= 0) {
      dropError = $t.invalidDrop;
      return;
    }
    drops = [...drops, { amount_cents: dropCents, note: dropNote.trim() }];
    dropCents = 0;
    dropNote = "";
  }

  function removeDrop(i: number) {
    const [removed] = drops.filter((_, j) => j === i);
    drops = drops.filter((_, j) => j !== i);
    toast($t.dropRemoved, {
      action: {
        label: $t.undo,
        onClick: () => {
          drops = [...drops.slice(0, i), removed, ...drops.slice(i)];
        },
      },
    });
  }

  function resetForm() {
    quantities = Object.fromEntries(denoms.map((d) => [d.id, 0]));
    denoms = denoms.map((d) => ({ ...d, qty: 0 }));
    expected = 0;
    expectedInvalid = false;
    expectedPick = "";
    drops = [];
    dropCents = 0;
    note = "";
    firstInput = true;
  }

  const saveDisabledReason = $derived(
    !isUrne && expectedError
      ? $t.saveFixExpected
      : counted === 0
        ? $t.saveNeedQty
        : "",
  );

  async function save() {
    if (saveDisabledReason || saving) return;
    saving = true;
    try {
      const items = denoms
        .filter((d) => d.qty > 0)
        .map((d) => ({ denomination_id: d.id, quantity: d.qty }));
      const res: any = await (CountSvc as any).CreateCount({
        type: countType,
        expected_cents: effectiveExpected,
        note,
        items,
        drops: drops.map((d) => ({
          amount_cents: d.amount_cents,
          note: d.note,
        })),
      });
      notifyDataChanged();
      refreshExisting();
      resetForm();
      toast.success(
        msg($t.countSavedDetail, {
          saved: msg($t.countSaved, { id: res.id }),
          variance: formatSignedCents(res.variance_cents, $currencySymbol, $locale),
        }),
        {
          action: {
            label: $t.viewInHistory,
            onClick: () => push(`/history/${res.id}`),
          },
        },
      );
    } catch (e) {
      toast.error($t.saveFailed, {
        description: e instanceof Error ? e.message : String(e),
      });
    } finally {
      saving = false;
    }
  }
</script>

<h1 class="mb-4 text-xl font-bold tracking-tight">{$t.countTitle}</h1>

{#if loading}
  <div class="flex flex-col gap-4" role="status" aria-label={$t.loading}>
    <Skeleton class="h-64 w-full" />
    <Skeleton class="h-40 w-full" />
  </div>
{:else if loadError}
  <Alert.Root variant="destructive">
    <CircleAlertIcon aria-hidden="true" />
    <Alert.Title>{$t.loadCountError}</Alert.Title>
    <Alert.Description>{loadError}</Alert.Description>
    <div class="mt-3">
      <Button variant="outline" size="sm" bind:ref={loadRetryBtn} onclick={load}>{$t.retry}</Button>
    </div>
  </Alert.Root>
{:else}
  <div class="grid items-start gap-4 lg:grid-cols-[1.4fr_1fr]">
    <Card.Root>
      <Card.Header>
        <Card.Title>{$t.denominations}</Card.Title>
        <Card.Description>{$t.denomHint}</Card.Description>
      </Card.Header>
      <Card.Content>
        {#if denoms.length === 0}
          <Empty.Root>
            <Empty.Header>
              <Empty.Media variant="icon"><WalletIcon aria-hidden="true" /></Empty.Media>
              <Empty.Title>{$t.noDenoms}</Empty.Title>
              <Empty.Description>{$t.noDenomsHint}</Empty.Description>
            </Empty.Header>
            <Empty.Content>
              <Button onclick={() => push("/settings")}>{$t.openSettings}</Button>
            </Empty.Content>
          </Empty.Root>
        {:else}
          {#each ["bill", "coin"] as kind}
            {@const group = denoms.filter((d) => d.kind === kind)}
            {#if group.length > 0}
              <p
                class="text-muted-foreground mt-4 mb-1 text-xs font-semibold tracking-wider uppercase first:mt-0"
              >
                {kind === "bill" ? $t.bills : $t.coins}
              </p>
              {#each group as d, di (d.id)}
                <DenomRow
                  denom={d}
                  qty={quantities[d.id] ?? 0}
                  onChange={(q) => setQty(d.id, q)}
                  autofocus={firstInput && kind === "bill" && di === 0}
                />
                {#if !(kind === "coin" && di === group.length - 1)}<Separator
                  />{/if}
              {/each}
            {/if}
          {/each}
          <div class="flex items-center justify-between pt-3 font-semibold">
            <span>{$t.totalCounted}</span>
            <AnimatedNumber value={counted} />
          </div>
        {/if}
      </Card.Content>
    </Card.Root>

    <div class="flex flex-col gap-4 self-start lg:sticky lg:top-20">
      <Card.Root>
        <Card.Header>
          <Card.Title>{$t.countDetails}</Card.Title>
        </Card.Header>
        <Card.Content>
          <Field.FieldGroup>
            <Field.Field>
              <Field.FieldLabel id="count-type-label"
                >{$t.countType}</Field.FieldLabel
              >
              <RadioGroup.Root
                bind:value={countType}
                aria-labelledby="count-type-label"
              >
                {#each typeOptions as o (o.value)}
                  {@const Icon = o.icon}
                  <div class="flex items-start gap-2.5">
                    <RadioGroup.Item
                      value={o.value}
                      id="count-type-{o.value}"
                      aria-label={o.label}
                      class="mt-0.5"
                    />
                    <div class="grid gap-0.5">
                      <Label
                        for="count-type-{o.value}"
                        class="flex items-center gap-1.5 font-medium"
                      >
                        <Icon aria-hidden="true" />{o.label}
                      </Label>
                      <p class="text-muted-foreground text-xs">{o.hint}</p>
                    </div>
                  </div>
                {/each}
              </RadioGroup.Root>
            </Field.Field>
            {#if isUrne}
              <p class="text-muted-foreground text-sm">
                {$t.expectedNotNeeded}
              </p>
            {:else if countedOptions.length > 0}
              <Field.Field>
                <Field.FieldLabel for="expected"
                  >{$t.expectedAmount}</Field.FieldLabel
                >
                <Select.Root
                  type="single"
                  bind:value={expectedPick}
                  onValueChange={pickExpected}
                >
                  <Select.Trigger id="expected" class="w-full tabular-nums">
                    {expectedPick && expectedPick !== CUSTOM
                      ? formatCents(
                          Number(expectedPick),
                          $currencySymbol,
                          $locale,
                        )
                      : $t.chooseExpected}
                  </Select.Trigger>
                  <Select.Content>
                    <Select.Group>
                      {#each countedOptions as o (o.amount)}
                        <Select.Item
                          value={String(o.amount)}
                          class="tabular-nums"
                        >
                          {formatCents(o.amount, $currencySymbol, $locale)}
                          <span class="text-muted-foreground">
                            · {fmtDate(o.last.created_at, $locale)} (#{o.last.id.slice(
                              -6,
                            )})
                          </span>
                        </Select.Item>
                      {/each}
                      <Select.Item value={CUSTOM}
                        >{$t.customAmount}</Select.Item
                      >
                    </Select.Group>
                  </Select.Content>
                </Select.Root>
              </Field.Field>
              {#if expectedPick === CUSTOM}
                <Field.Field data-invalid={expectedError ? true : undefined}>
                  <Field.FieldLabel for="expected-custom"
                    >{$t.customAmount}</Field.FieldLabel
                  >
                  <CentsInput
                    id="expected-custom"
                    bind:cents={expected}
                    bind:invalid={expectedInvalid}
                    placeholder={zeroPlaceholder}
                    ariaLabel={$t.customAmount}
                    onRevert={() => {
                      expectedInvalid = true;
                    }}
                  />
                  {#if expectedError}<Field.FieldError
                      >{expectedError}</Field.FieldError
                    >{/if}
                </Field.Field>
              {/if}
            {:else}
              <Field.Field data-invalid={expectedError ? true : undefined}>
                <Field.FieldLabel for="expected"
                  >{$t.expectedAmount}</Field.FieldLabel
                >
                <CentsInput
                  id="expected"
                  bind:cents={expected}
                  bind:invalid={expectedInvalid}
                  placeholder={zeroPlaceholder}
                  ariaLabel={$t.expectedAmount}
                  onRevert={() => {
                    expectedInvalid = true;
                  }}
                />
                {#if expectedError}<Field.FieldError
                    >{expectedError}</Field.FieldError
                  >{/if}
              </Field.Field>
            {/if}
            <Field.Field>
              <Field.FieldLabel for="note">{$t.noteOptional}</Field.FieldLabel>
              <Textarea
                id="note"
                bind:value={note}
                placeholder={$t.notePlaceholder}
              />
            </Field.Field>
          </Field.FieldGroup>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header>
          <Card.Title>{$t.safeDrops}</Card.Title>
          <Card.Description
            >{$t.dropsHint} (<AnimatedNumber
              value={dropsTotal}
            />).</Card.Description
          >
        </Card.Header>
        <Card.Content class="flex flex-col gap-3">
          {#each drops as d, i}
            <div class="flex items-center gap-2 text-sm">
              <strong><AnimatedNumber value={d.amount_cents} /></strong>
              <span class="text-muted-foreground truncate" title={d.note || undefined}
                >{d.note || $t.dropFallbackNote}</span
              >
              <span class="flex-1"></span>
              <Button variant="ghost" size="sm" class="min-h-9 min-w-9" onclick={() => removeDrop(i)}
                >{$t.remove}</Button
              >
            </div>
            <Separator />
          {/each}
          <div class="flex items-end gap-2">
            <Field.Field
              class="flex-1"
              data-invalid={dropError ? true : undefined}
            >
              <Field.FieldLabel for="drop-amount" class="sr-only"
                >{$t.dropAmount}</Field.FieldLabel
              >
              <CentsInput
                id="drop-amount"
                bind:cents={dropCents}
                bind:invalid={dropInvalid}
                placeholder={$t.dropAmount}
                ariaLabel={$t.dropAmount}
                describedBy={dropError ? "drop-error" : undefined}
                onCommit={addDrop}
                onRevert={() => {
                  dropError = $t.invalidDrop;
                }}
              />
            </Field.Field>
            <Field.Field class="flex-2">
              <Field.FieldLabel for="drop-note" class="sr-only"
                >{$t.dropNote}</Field.FieldLabel
              >
              <Input
                id="drop-note"
                placeholder={$t.dropNote}
                bind:value={dropNote}
                onkeydown={(e) =>
                  e.key === "Enter" && (e.preventDefault(), addDrop())}
              />
            </Field.Field>
            <Button
              variant="outline"
              size="icon"
              onclick={addDrop}
              aria-label={$t.addDrop}
            >
              <PlusIcon aria-hidden="true" />
            </Button>
          </div>
          {#if dropError}<p id="drop-error" role="alert" class="text-destructive text-sm">
              {dropError}
            </p>{/if}
        </Card.Content>
      </Card.Root>

      <VariancePanel {variance} />
      <div class="grid">
        <Button
          class="w-full"
          size="lg"
          disabled={saving || !!saveDisabledReason}
          onclick={save}
          aria-describedby={saveDisabledReason ? "save-blocked-reason" : undefined}
        >
          <span class="grid items-center justify-items-center">
            <span class="col-start-1 row-start-1 {saving ? 'invisible' : ''}"
              >{$t.saveCount}</span
            >
            {#if saving}
              <span class="col-start-1 row-start-1 flex items-center gap-2"
                ><Spinner />{$t.saving}</span
              >
            {/if}
          </span>
        </Button>
      </div>
      {#if saveDisabledReason && !saving}
        <p id="save-blocked-reason" role="status" class="text-muted-foreground text-sm">
          {saveDisabledReason}
        </p>
      {/if}
    </div>
  </div>
{/if}
