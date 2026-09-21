<script lang="ts">
  import { onMount } from "svelte";
  import { link, push } from "svelte-spa-router";
  import { toast } from "svelte-sonner";
  import { Clipboard } from "@wailsio/runtime";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import ReceiptIcon from "@lucide/svelte/icons/receipt";
  import * as Card from "$lib/components/ui/card";
  import * as Field from "$lib/components/ui/field";
  import * as Select from "$lib/components/ui/select";
  import * as Table from "$lib/components/ui/table";
  import * as Empty from "$lib/components/ui/empty";
  import * as Alert from "$lib/components/ui/alert";
  import { Badge } from "$lib/components/ui/badge";
  import { Button } from "$lib/components/ui/button";
  import { Skeleton } from "$lib/components/ui/skeleton";
  import { Spinner } from "$lib/components/ui/spinner";
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import CountTypeBadge from "../lib/CountTypeBadge.svelte";
  import { formatCents, formatSignedCents } from "../lib/money";
  import { fmtDate, countTypeLabel } from "../lib/format";
  import { currencySymbol } from "../lib/ui";
  import { t, msg, locale } from "../lib/i18n";
  import { dataVersion } from "../lib/state";
  import { CountSvc } from "../lib/services";
  import type { CountSummary } from "../lib/types";

  let counts = $state<CountSummary[]>([]);
  let loading = $state(true);
  let loadError = $state("");
  let typeFilter = $state("all");
  let exporting = $state(false);
  let loadRetryBtn: HTMLElement | null = $state(null);
  let lastLoadError = $state("");

  const isFiltered = $derived(typeFilter !== "all");

  const typeOptions = $derived([
    { value: "all", label: $t.allTypes },
    { value: "shift_open", label: $t.typeOpen },
    { value: "shift_close", label: $t.typeClose },
    { value: "audit", label: $t.typeAudit },
    { value: "donation_urne", label: $t.typeDonationUrne },
  ]);

  onMount(() => {
    const unsub = dataVersion.subscribe(() => load());
    load();
    return unsub;
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
      const filter: any = {};
      if (typeFilter && typeFilter !== "all") filter.type = typeFilter;
      counts = (await (CountSvc as any).ListCounts(filter)) as CountSummary[];
    } catch (e) {
      loadError = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  function clearFilter() {
    typeFilter = "all";
    load();
  }

  async function exportHistory() {
    exporting = true;
    try {
      const path = (await (CountSvc as any).ExportHistoryCSV()) as string;
      toast.success($t.historyExported, {
        description: path,
        action: {
          label: $t.copyPath,
          onClick: () => {
            Clipboard.SetText(path).catch((e: unknown) =>
              toast.error(e instanceof Error ? e.message : String(e)),
            );
          },
        },
      });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      exporting = false;
    }
  }
</script>

<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
  <h1 class="text-xl font-bold tracking-tight">{$t.historyTitle}</h1>
  <Button variant="outline" size="sm" disabled={exporting || counts.length === 0} onclick={exportHistory}>
    <span class="grid items-center justify-items-center">
      <span class="col-start-1 row-start-1 {exporting ? 'invisible' : ''} flex items-center gap-1.5">
        <DownloadIcon data-icon="inline-start" aria-hidden="true" />{$t.exportAll}
      </span>
      {#if exporting}
        <span class="col-start-1 row-start-1 flex items-center gap-2"><Spinner />{$t.exporting}</span>
      {/if}
    </span>
  </Button>
</div>

{#if loadError}
  <Alert.Root variant="destructive">
    <CircleAlertIcon aria-hidden="true" />
    <Alert.Title>{$t.loadHistoryError}</Alert.Title>
    <Alert.Description>{loadError}</Alert.Description>
    <div class="mt-3">
      <Button variant="outline" size="sm" bind:ref={loadRetryBtn} onclick={load}>{$t.retry}</Button>
    </div>
  </Alert.Root>
{:else}
  <Card.Root>
    <Card.Header>
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <Card.Title>{$t.counts}</Card.Title>
          <Card.Description>{$t.countsHint}</Card.Description>
        </div>
        <Field.Field class="w-44">
          <Field.FieldLabel for="type-filter">{$t.filterByType}</Field.FieldLabel>
          <Select.Root type="single" bind:value={typeFilter} onValueChange={load}>
            <Select.Trigger id="type-filter" class="w-44">
              {typeOptions.find((o) => o.value === typeFilter)?.label ?? $t.allTypes}
            </Select.Trigger>
            <Select.Content>
              <Select.Group>
                {#each typeOptions as o (o.value)}
                  <Select.Item value={o.value}>{o.label}</Select.Item>
                {/each}
              </Select.Group>
            </Select.Content>
          </Select.Root>
        </Field.Field>
      </div>
      {#if !loading && !loadError && (counts.length > 0 || isFiltered)}
        <p class="text-muted-foreground mt-2 text-xs" role="status">
          {msg($t.resultCount, { n: counts.length })}
        </p>
      {/if}
    </Card.Header>
    <Card.Content>
      {#if loading}
        <div class="flex flex-col gap-2" role="status" aria-label={$t.loading}>
          <Skeleton class="h-9 w-full" />
          <Skeleton class="h-9 w-full" />
          <Skeleton class="h-9 w-full" />
        </div>
      {:else if counts.length === 0 && isFiltered}
        <Empty.Root>
          <Empty.Header>
            <Empty.Media variant="icon"><ReceiptIcon aria-hidden="true" /></Empty.Media>
            <Empty.Title>{$t.noResults}</Empty.Title>
            <Empty.Description>{$t.noResultsHint}</Empty.Description>
          </Empty.Header>
          <Empty.Content>
            <Button variant="outline" onclick={clearFilter}>{$t.clearFilter}</Button>
          </Empty.Content>
        </Empty.Root>
      {:else if counts.length === 0}
        <Empty.Root>
          <Empty.Header>
            <Empty.Media variant="icon"><ReceiptIcon aria-hidden="true" /></Empty.Media>
            <Empty.Title>{$t.noCounts}</Empty.Title>
            <Empty.Description>{$t.noCountsHint}</Empty.Description>
          </Empty.Header>
          <Empty.Content>
            <Button onclick={() => push("/")}>{$t.newCount}</Button>
          </Empty.Content>
        </Empty.Root>
      {:else}
        <div class="overflow-x-auto">
          <Table.Root>
            <Table.Header>
              <Table.Row>
                <Table.Head>{$t.date}</Table.Head>
                <Table.Head>{$t.type}</Table.Head>
                <Table.Head class="text-right">{$t.counted}</Table.Head>
                <Table.Head class="text-right">{$t.expected}</Table.Head>
                <Table.Head class="text-right">{$t.varianceCol}</Table.Head>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {#each counts as c (c.id)}
                {@const varianceLabel = c.variance_cents > 0 ? $t.varianceOver : c.variance_cents < 0 ? $t.varianceShort : $t.varianceBalanced}
                <Table.Row class="hover:bg-muted/50">
                  <Table.Cell class="whitespace-nowrap">
                    <a
                      href="#/history/{c.id}"
                      use:link
                      class="font-medium underline-offset-4 hover:underline"
                      aria-label={msg($t.openCountDetail, {
                        date: fmtDate(c.created_at, $locale),
                        type: countTypeLabel(c.type, $t),
                        variance: formatSignedCents(c.variance_cents, $currencySymbol, $locale),
                      })}
                    >
                      {fmtDate(c.created_at, $locale)}
                    </a>
                  </Table.Cell>
                  <Table.Cell><CountTypeBadge type={c.type} /></Table.Cell>
                  <Table.Cell class="text-right tabular-nums">{formatCents(c.counted_cents, $currencySymbol, $locale)}</Table.Cell>
                  <Table.Cell class="text-right tabular-nums">{formatCents(c.expected_cents, $currencySymbol, $locale)}</Table.Cell>
                  <Table.Cell class="text-right">
                    {#if c.variance_cents > 0}
                      <Badge class="bg-success text-success-foreground hover:bg-success/90 tabular-nums"><span class="sr-only">{varianceLabel} </span>{formatSignedCents(c.variance_cents, $currencySymbol, $locale)}</Badge>
                    {:else if c.variance_cents < 0}
                      <Badge variant="destructive" class="tabular-nums"><span class="sr-only">{varianceLabel} </span>{formatSignedCents(c.variance_cents, $currencySymbol, $locale)}</Badge>
                    {:else}
                      <Badge variant="secondary" class="tabular-nums"><span class="sr-only">{varianceLabel} </span>{formatSignedCents(c.variance_cents, $currencySymbol, $locale)}</Badge>
                    {/if}
                  </Table.Cell>
                </Table.Row>
              {/each}
            </Table.Body>
          </Table.Root>
        </div>
      {/if}
    </Card.Content>
  </Card.Root>
{/if}
