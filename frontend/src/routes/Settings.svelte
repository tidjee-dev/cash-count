<script lang="ts">
  import { toast } from "svelte-sonner";
  import PlusIcon from "@lucide/svelte/icons/plus";
  import XIcon from "@lucide/svelte/icons/x";
  import * as Card from "$lib/components/ui/card";
  import * as Field from "$lib/components/ui/field";
  import * as Select from "$lib/components/ui/select";
  import * as Alert from "$lib/components/ui/alert";
  import * as AlertDialog from "$lib/components/ui/alert-dialog";
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import { Separator } from "$lib/components/ui/separator";
  import { Skeleton } from "$lib/components/ui/skeleton";
  import { Spinner } from "$lib/components/ui/spinner";
  import { Switch } from "$lib/components/ui/switch";
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import { settings, denominations, notifyDataChanged } from "../lib/state";
  import { loadAppData } from "../lib/backend";
  import { SettingsSvc, DenominationSvc } from "../lib/services";
  import { t, msg } from "../lib/i18n";
  import CentsInput from "../lib/CentsInput.svelte";
  import type { Denomination } from "../lib/types";

  interface Preset {
    label: string;
    symbol: string;
    denominations: { label: string; value_cents: number; kind: string }[];
  }

  let loading = $state(true);
  let loadError = $state("");
  let storeName = $state("");
  let currencyLabel = $state("EUR");
  let currencySymbolVal = $state("€");
  let presets = $state<Preset[]>([]);
  let rows = $state<(Denomination & { _deleted?: boolean })[]>([]);
  let savingSettings = $state(false);
  let savingDenoms = $state(false);
  let settingsErrors = $state<{ label: string; symbol: string }>({ label: "", symbol: "" });
  let denomsError = $state("");
  let loadRetryBtn: HTMLElement | null = $state(null);
  let lastLoadError = $state("");
  let newLabelEl: HTMLInputElement | null = $state(null);

  let nl = $state("");
  let newCents = $state(0);
  let newInvalid = $state(false);
  let nk = $state("bill");
  let nerr = $state("");
  let pendingPreset: Preset | null = $state(null);
  let confirmPreset = $state(false);
  let presetCancelBtn: HTMLElement | null = $state(null);
  let savedDenomsKey = $state("");

  const denomsKey = $derived(
    JSON.stringify(
      rows
        .filter((r) => !r._deleted)
        .map((r) => [r.id, r.label, r.value_cents, r.kind, r.active]),
    ),
  );
  const denomsDirty = $derived(denomsKey !== savedDenomsKey);

  $effect(() => {
    load();
  });

  async function load() {
    loading = true;
    loadError = "";
    try {
      const data = await loadAppData();
      storeName = data.settings.store_name;
      currencyLabel = data.settings.currency_label;
      currencySymbolVal = data.settings.currency_symbol;
      settingsErrors = { label: "", symbol: "" };
      presets = (await (SettingsSvc as any).ListCurrencyPresets()) as Preset[];
      rows = await (DenominationSvc as any).List(false);
      savedDenomsKey = JSON.stringify(
        rows.map((r: Denomination) => [r.id, r.label, r.value_cents, r.kind, r.active]),
      );
    } catch (e) {
      loadError = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  // Move focus to Retry only when a load failure freshly appears.
  $effect(() => {
    if (loadError && loadError !== lastLoadError) {
      lastLoadError = loadError;
      loadRetryBtn?.focus();
    } else if (!loadError) {
      lastLoadError = "";
    }
  });

  async function saveSettings() {
    settingsErrors = {
      label: currencyLabel.trim() ? "" : $t.needCurrencyLabel,
      symbol: currencySymbolVal.trim() ? "" : $t.needCurrencySymbol,
    };
    if (settingsErrors.label || settingsErrors.symbol) return;
    savingSettings = true;
    try {
      const s = await (SettingsSvc as any).SaveSettings({
        store_name: storeName,
        currency_label: currencyLabel,
        currency_symbol: currencySymbolVal,
      });
      settings.set(s);
      notifyDataChanged();
      toast.success($t.settingsSaved);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      savingSettings = false;
    }
  }

  async function applyPreset(p: Preset) {
    pendingPreset = null;
    confirmPreset = false;
    currencyLabel = p.label;
    currencySymbolVal = p.symbol;
    savingSettings = true;
    try {
      const s = await (SettingsSvc as any).SaveSettings({
        store_name: storeName,
        currency_label: currencyLabel,
        currency_symbol: currencySymbolVal,
      });
      settings.set(s);
      rows = p.denominations.map((d, i) => ({
        id: "",
        label: d.label,
        value_cents: d.value_cents,
        kind: d.kind,
        sort_order: i,
        active: true,
      }));
      await saveDenoms();
      await load();
      toast.success(msg($t.presetApplied, { label: p.label }));
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      savingSettings = false;
    }
  }

  function addRow() {
    nerr = "";
    if (!nl.trim()) {
      nerr = $t.needLabel;
      return;
    }
    if (newInvalid || newCents <= 0) {
      nerr = $t.needValue;
      return;
    }
    rows = [...rows, { id: "", label: nl.trim(), value_cents: newCents, kind: nk, sort_order: rows.length, active: true }];
    nl = "";
    newCents = 0;
    // Keep the rapid-entry flow: focus returns to the label field.
    newLabelEl?.focus();
  }

  function removeRow(index: number) {
    const removed = rows[index];
    rows = rows.map((r, i) => (i === index ? { ...r, _deleted: true } : r));
    toast($t.denomRemoved, {
      action: {
        label: $t.undo,
        onClick: () => {
          rows = rows.map((r) => (r === rows[index] ? removed : r));
        },
      },
    });
  }

  async function saveDenoms() {
    denomsError = "";
    const visible = rows.filter((r) => !r._deleted);
    const bad = visible.findIndex((r) => !r.label.trim() || r.value_cents <= 0);
    if (bad >= 0) {
      denomsError = msg($t.denomsInvalid, { n: bad + 1 });
      return;
    }
    savingDenoms = true;
    try {
      const payload = visible
        .map((r, i) => ({ id: r.id, label: r.label, value_cents: r.value_cents, kind: r.kind, sort_order: i, active: r.active }));
      const list = (await (DenominationSvc as any).SaveAll(payload)) as Denomination[];
      denominations.set(list.filter((d) => d.active));
      rows = await (DenominationSvc as any).List(false);
      savedDenomsKey = JSON.stringify(
        rows.map((r: Denomination) => [r.id, r.label, r.value_cents, r.kind, r.active]),
      );
      notifyDataChanged();
      toast.success($t.denomsSaved);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      savingDenoms = false;
    }
  }
</script>

<h1 class="mb-4 text-xl font-bold tracking-tight">{$t.settingsTitle}</h1>

{#if loading}
  <div class="flex flex-col gap-4" role="status" aria-label={$t.loading}>
    <Skeleton class="h-56 w-full" />
    <Skeleton class="h-56 w-full" />
  </div>
{:else if loadError}
  <Alert.Root variant="destructive">
    <CircleAlertIcon aria-hidden="true" />
    <Alert.Title>{$t.loadSettingsError}</Alert.Title>
    <Alert.Description>{loadError}</Alert.Description>
    <div class="mt-3">
      <Button variant="outline" size="sm" bind:ref={loadRetryBtn} onclick={load}>{$t.retry}</Button>
    </div>
  </Alert.Root>
{:else}
  <div class="grid items-start gap-4 lg:grid-cols-2">
    <Card.Root>
      <Card.Header>
        <Card.Title>{$t.storeCurrency}</Card.Title>
        <Card.Description>{$t.storeCurrencyHint}</Card.Description>
      </Card.Header>
      <Card.Content>
        <Field.FieldGroup>
          <Field.Field>
            <Field.FieldLabel for="store-name">{$t.storeName}</Field.FieldLabel>
            <Field.FieldDescription>{$t.storeNameHint}</Field.FieldDescription>
            <Input id="store-name" bind:value={storeName} placeholder={$t.storeNamePlaceholder} class="max-w-56" />
          </Field.Field>
          <Field.Field data-invalid={settingsErrors.label ? true : undefined}>
            <Field.FieldLabel for="currency-label">{$t.currencyLabel}</Field.FieldLabel>
            <Input id="currency-label" bind:value={currencyLabel} placeholder="EUR" class="max-w-28" aria-describedby={settingsErrors.label ? "currency-label-error" : undefined} />
            {#if settingsErrors.label}<Field.FieldError id="currency-label-error">{settingsErrors.label}</Field.FieldError>{/if}
          </Field.Field>
          <Field.Field data-invalid={settingsErrors.symbol ? true : undefined}>
            <Field.FieldLabel for="currency-symbol">{$t.currencySymbol}</Field.FieldLabel>
            <Input id="currency-symbol" bind:value={currencySymbolVal} placeholder="€" class="max-w-20" aria-describedby={settingsErrors.symbol ? "currency-symbol-error" : undefined} />
            {#if settingsErrors.symbol}<Field.FieldError id="currency-symbol-error">{settingsErrors.symbol}</Field.FieldError>{/if}
          </Field.Field>
        </Field.FieldGroup>
      </Card.Content>
      <Card.Footer>
        <Button disabled={savingSettings} onclick={saveSettings}>
          <span class="grid items-center justify-items-center">
            <span class="col-start-1 row-start-1 {savingSettings ? 'invisible' : ''}">{$t.saveSettings}</span>
            {#if savingSettings}
              <span class="col-start-1 row-start-1 flex items-center gap-2"><Spinner />{$t.saving}</span>
            {/if}
          </span>
        </Button>
      </Card.Footer>
    </Card.Root>

    <Card.Root>
      <Card.Header>
        <Card.Title>{$t.currencyPresets}</Card.Title>
        <Card.Description>{$t.currencyPresetsHint}</Card.Description>
      </Card.Header>
      <Card.Content class="flex flex-wrap gap-2">
        {#each presets as p}
          <Button variant="outline" size="sm" disabled={savingSettings || savingDenoms} onclick={() => { pendingPreset = p; confirmPreset = true; }}>
            {p.label} ({p.symbol})
          </Button>
        {/each}
      </Card.Content>
    </Card.Root>

    <Card.Root class="lg:col-span-2">
      <Card.Header>
        <Card.Title>{$t.denominationsTitle}</Card.Title>
        <Card.Description>{$t.denominationsHint}</Card.Description>
      </Card.Header>
      <Card.Content class="flex flex-col gap-2">
        {#if denomsError}<p role="alert" class="text-destructive text-sm">{denomsError}</p>{/if}
        <div class="text-muted-foreground hidden gap-2 text-xs font-medium sm:grid sm:grid-cols-[9rem_7rem_7rem_1fr_auto]" aria-hidden="true">
          <span>{$t.newLabel}</span>
          <span>{$t.newValue}</span>
          <span>{$t.denomKindAria}</span>
          <span>{$t.colActive}</span>
          <span class="sr-only">{$t.colActions}</span>
        </div>
        {#each rows.map((r, i) => ({ r, i })).filter(({ r }) => !r._deleted) as { r, i } (r.id || `${r.label}-${r.value_cents}`)}
          <div class="flex flex-wrap items-center gap-2">
            <Input bind:value={r.label} aria-label={$t.denomLabelAria} class="max-w-36" />
            <CentsInput
              bind:cents={r.value_cents}
              ariaLabel={$t.denomValueAria}
              hideZero={false}
              class="max-w-28 text-right"
            />
            <Select.Root type="single" bind:value={r.kind}>
              <Select.Trigger aria-label={$t.denomKindAria} class="w-28">
                {r.kind === "coin" ? $t.coin : $t.bill}
              </Select.Trigger>
              <Select.Content>
                <Select.Group>
                  <Select.Item value="bill">{$t.bill}</Select.Item>
                  <Select.Item value="coin">{$t.coin}</Select.Item>
                </Select.Group>
              </Select.Content>
            </Select.Root>
            <span class="flex items-center gap-2">
              <Switch bind:checked={r.active} aria-label={msg($t.activeOf, { label: r.label })} />
              <span class="text-muted-foreground text-xs sm:hidden">{$t.colActive}</span>
            </span>
            <Button variant="ghost" size="icon-lg" onclick={() => removeRow(i)} aria-label={msg($t.removeOf, { label: r.label })}>
              <XIcon aria-hidden="true" />
            </Button>
          </div>
          <Separator />
        {/each}
        <fieldset class="flex flex-wrap items-end gap-2 pt-1">
          <legend class="sr-only">{$t.addDenomLegend}</legend>
          <Field.Field class="max-w-36">
            <Field.FieldLabel for="new-label" class="sr-only">{$t.newLabel}</Field.FieldLabel>
            <Input
              id="new-label"
              placeholder={$t.newLabel}
              bind:value={nl}
              bind:ref={newLabelEl}
              aria-describedby={nerr ? "new-row-error" : undefined}
              onkeydown={(e) => e.key === "Enter" && (e.preventDefault(), addRow())}
            />
          </Field.Field>
          <Field.Field class="max-w-28">
            <Field.FieldLabel for="new-value" class="sr-only">{$t.newValue}</Field.FieldLabel>
            <CentsInput
              id="new-value"
              bind:cents={newCents}
              bind:invalid={newInvalid}
              placeholder={$t.newValue}
              ariaLabel={$t.newValue}
              describedBy={nerr ? "new-row-error" : undefined}
              class="text-right"
              onCommit={addRow}
              onRevert={() => {
                nerr = $t.needValue;
              }}
            />
          </Field.Field>
          <Select.Root type="single" bind:value={nk}>
            <Select.Trigger aria-label={$t.newKindAria} class="w-28">{nk === "coin" ? $t.coin : $t.bill}</Select.Trigger>
            <Select.Content>
              <Select.Group>
                <Select.Item value="bill">{$t.bill}</Select.Item>
                <Select.Item value="coin">{$t.coin}</Select.Item>
              </Select.Group>
            </Select.Content>
          </Select.Root>
          <Button variant="outline" onclick={addRow}>
            <PlusIcon data-icon="inline-start" aria-hidden="true" />{$t.add}
          </Button>
        </fieldset>
        {#if nerr}<p id="new-row-error" role="alert" class="text-destructive text-sm">{nerr}</p>{/if}
      </Card.Content>
      <Card.Footer class="gap-3">
        <Button disabled={savingDenoms || !denomsDirty} onclick={saveDenoms}>
          <span class="grid items-center justify-items-center">
            <span class="col-start-1 row-start-1 {savingDenoms ? 'invisible' : ''}">{$t.saveDenoms}</span>
            {#if savingDenoms}
              <span class="col-start-1 row-start-1 flex items-center gap-2"><Spinner />{$t.saving}</span>
            {/if}
          </span>
        </Button>
        {#if !denomsDirty}<span class="text-muted-foreground text-xs">{$t.noUnsavedChanges}</span>{/if}
      </Card.Footer>
    </Card.Root>
  </div>
{/if}

<AlertDialog.Root bind:open={confirmPreset}>
  <AlertDialog.Content
    onOpenAutoFocus={(e) => {
      e.preventDefault();
      presetCancelBtn?.focus();
    }}
  >
    <AlertDialog.Header>
      <AlertDialog.Title>{pendingPreset ? msg($t.presetConfirmTitle, { label: pendingPreset.label }) : ""}</AlertDialog.Title>
      <AlertDialog.Description>
        {pendingPreset ? msg($t.presetConfirmBody, { label: pendingPreset.label }) : ""}
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer>
      <AlertDialog.Cancel bind:ref={presetCancelBtn} disabled={savingSettings} onclick={() => (pendingPreset = null)}>{$t.cancel}</AlertDialog.Cancel>
      <AlertDialog.Action disabled={savingSettings || !pendingPreset} onclick={() => pendingPreset && applyPreset(pendingPreset)}>
        <span class="grid items-center justify-items-center">
          <span class="col-start-1 row-start-1 {savingSettings ? 'invisible' : ''}">{$t.confirm}</span>
          {#if savingSettings}
            <span class="col-start-1 row-start-1 flex items-center gap-2"><Spinner />{$t.saving}</span>
          {/if}
        </span>
      </AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
