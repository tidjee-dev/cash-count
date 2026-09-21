<script lang="ts">
  import { push } from "svelte-spa-router";
  import { toast } from "svelte-sonner";
  import { Clipboard } from "@wailsio/runtime";
  import ArrowLeftIcon from "@lucide/svelte/icons/arrow-left";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import * as Card from "$lib/components/ui/card";
  import * as Table from "$lib/components/ui/table";
  import * as Alert from "$lib/components/ui/alert";
  import * as AlertDialog from "$lib/components/ui/alert-dialog";
  import { Badge } from "$lib/components/ui/badge";
  import { Button, buttonVariants } from "$lib/components/ui/button";
  import { Skeleton } from "$lib/components/ui/skeleton";
  import { Spinner } from "$lib/components/ui/spinner";
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import CountTypeBadge from "../lib/CountTypeBadge.svelte";
  import { formatCents, formatSignedCents, varianceState } from "../lib/money";
  import { fmtDate, countTypeLabel } from "../lib/format";
  import { currencySymbol } from "../lib/ui";
  import { t, msg, locale } from "../lib/i18n";
  import { notifyDataChanged } from "../lib/state";
  import { CountSvc } from "../lib/services";
  import type { CountDetail } from "../lib/types";

  let { params = {} }: { params?: { id?: string } } = $props();
  const id = $derived(params.id ?? "");

  let detail = $state<CountDetail | null>(null);
  let loading = $state(true);
  let loadError = $state("");
  let confirmDelete = $state(false);
  let deleting = $state(false);
  let exporting = $state(false);
  let cancelBtn: HTMLElement | null = $state(null);
  let deleteBtn: HTMLElement | null = $state(null);
  let loadRetryBtn: HTMLElement | null = $state(null);
  let lastLoadError = $state("");
  let wasConfirmOpen = $state(false);

  const varianceText = $derived(
    detail === null
      ? ""
      : (() => {
          const s = varianceState(detail.variance_cents);
          return s === "over" ? $t.varianceOver : s === "short" ? $t.varianceShort : $t.varianceBalanced;
        })()
  );

  $effect(() => {
    if (id !== "") load();
    else {
      loading = false;
      detail = null;
      loadError = $t.invalidCountId;
    }
  });

  // Return focus to the Delete trigger when the dialog closes.
  $effect(() => {
    if (wasConfirmOpen && !confirmDelete) deleteBtn?.focus();
    wasConfirmOpen = confirmDelete;
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
    detail = null;
    try {
      detail = (await (CountSvc as any).GetCount(id)) as CountDetail;
    } catch (e) {
      loadError = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  async function remove() {
    const snapshot = detail;
    deleting = true;
    try {
      await (CountSvc as any).DeleteCount(id);
      notifyDataChanged();
      confirmDelete = false;
      toast.success(msg($t.countDeleted, { id: `…${id.slice(-6)}` }), {
        action: snapshot
          ? {
              label: $t.undo,
              onClick: async () => {
                try {
                  await (CountSvc as any).RestoreCount(snapshot);
                  notifyDataChanged();
                  push(`/history/${snapshot.id}`);
                } catch (e) {
                  toast.error(e instanceof Error ? e.message : String(e));
                }
              },
            }
          : undefined,
      });
      push("/history");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      deleting = false;
    }
  }

  async function exportOne() {
    exporting = true;
    try {
      const path = (await (CountSvc as any).ExportCountCSV(id)) as string;
      toast.success($t.countExported, {
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

<a href="#/history" class={buttonVariants({ variant: "ghost", size: "sm" })}>
  <ArrowLeftIcon data-icon="inline-start" aria-hidden="true" />{$t.backToHistory}
</a>

{#if loading}
  <div class="mt-4 flex flex-col gap-4" role="status" aria-label={$t.loading}>
    <Skeleton class="h-8 w-64" />
    <Skeleton class="h-64 w-full" />
  </div>
{:else if loadError && !detail}
  <Alert.Root variant="destructive" class="mt-4">
    <CircleAlertIcon aria-hidden="true" />
    <Alert.Title>{$t.loadDetailError}</Alert.Title>
    <Alert.Description>{loadError}</Alert.Description>
    {#if id}
      <div class="mt-3">
        <Button variant="outline" size="sm" bind:ref={loadRetryBtn} onclick={load}>{$t.retry}</Button>
      </div>
    {/if}
  </Alert.Root>
{:else if !detail}
  <Alert.Root variant="destructive" class="mt-4">
    <CircleAlertIcon aria-hidden="true" />
    <Alert.Title>{$t.loadDetailError}</Alert.Title>
    <Alert.Description>{$t.invalidCountId}</Alert.Description>
  </Alert.Root>
{:else if detail}
  <div class="mt-4 mb-4 flex flex-wrap items-center justify-between gap-3">
    <h1 class="text-xl font-bold tracking-tight" title={detail.id}>
      {msg($t.detailTitle, { id: `…${detail.id.slice(-6)}` })}
      <span class="ml-2 inline-block align-middle"><CountTypeBadge type={detail.type} /></span>
    </h1>
    <div class="flex items-center gap-2">
      <span class="grid">
        <Button variant="outline" size="sm" disabled={exporting} onclick={exportOne}>
          <span class="grid items-center justify-items-center">
            <span class="col-start-1 row-start-1 {exporting ? 'invisible' : ''} flex items-center gap-1.5">
              <DownloadIcon data-icon="inline-start" aria-hidden="true" />{$t.exportCsv}
            </span>
            {#if exporting}
              <span class="col-start-1 row-start-1 flex items-center gap-2"><Spinner />{$t.exporting}</span>
            {/if}
          </span>
        </Button>
      </span>
      <Button variant="destructive" size="sm" bind:ref={deleteBtn} onclick={() => (confirmDelete = true)}>
        <Trash2Icon data-icon="inline-start" aria-hidden="true" />{$t.delete}
      </Button>
    </div>
  </div>

  <div class="grid items-start gap-4 lg:grid-cols-[1.4fr_1fr]">
    <Card.Root class="min-w-0">
      <Card.Header>
        <Card.Title>{$t.breakdown}</Card.Title>
      </Card.Header>
      <Card.Content>
        {#if detail.items.length === 0}
          <p class="text-muted-foreground text-sm">{$t.noBreakdown}</p>
        {:else}
        <div class="overflow-x-auto">
        <Table.Root>
          <Table.Header>
            <Table.Row>
              <Table.Head>{$t.denomination}</Table.Head>
              <Table.Head class="text-right">{$t.qty}</Table.Head>
              <Table.Head class="text-right">{$t.subtotal}</Table.Head>
            </Table.Row>
          </Table.Header>
          <Table.Body>
            {#each detail.items as it (it.id)}
              <Table.Row>
                <Table.Cell>{it.denomination_label}</Table.Cell>
                <Table.Cell class="text-right tabular-nums">{it.quantity}</Table.Cell>
                <Table.Cell class="text-right tabular-nums">{formatCents(it.subtotal_cents, $currencySymbol, $locale)}</Table.Cell>
              </Table.Row>
            {/each}
          </Table.Body>
        </Table.Root>
        </div>
        {/if}
        {#if detail.drops.length > 0}
          <h2 class="mt-6 mb-2 text-sm font-semibold">{$t.safeDrops}</h2>
          <div class="overflow-x-auto">
          <Table.Root>
            <Table.Header>
              <Table.Row>
                <Table.Head>{$t.dropNote}</Table.Head>
                <Table.Head class="text-right">{$t.dropAmount}</Table.Head>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {#each detail.drops as d (d.id)}
                <Table.Row>
                  <Table.Cell>{d.note || $t.dropFallbackNote}</Table.Cell>
                  <Table.Cell class="text-right tabular-nums">{formatCents(d.amount_cents, $currencySymbol, $locale)}</Table.Cell>
                </Table.Row>
              {/each}
            </Table.Body>
          </Table.Root>
          </div>
        {/if}
        {#if detail.note}<p class="text-muted-foreground mt-4 text-sm">{msg($t.noteLabel, { note: detail.note })}</p>{/if}
      </Card.Content>
    </Card.Root>
    <Card.Root class="order-first min-w-0 self-start lg:order-none lg:sticky lg:top-20">
      <Card.Header>
        <Card.Title>{$t.totals}</Card.Title>
        <Card.Description>{msg($t.recordedAt, { date: fmtDate(detail.created_at, $locale) })}</Card.Description>
      </Card.Header>
      <Card.Content>
        <div class="overflow-x-auto">
        <Table.Root>
          <Table.Body>
            <Table.Row><Table.Cell>{$t.counted}</Table.Cell><Table.Cell class="text-right tabular-nums">{formatCents(detail.counted_cents, $currencySymbol, $locale)}</Table.Cell></Table.Row>
            <Table.Row><Table.Cell>{$t.safeDrops}</Table.Cell><Table.Cell class="text-right tabular-nums">{formatCents(detail.drops_cents, $currencySymbol, $locale)}</Table.Cell></Table.Row>
            <Table.Row><Table.Cell>{$t.expected}</Table.Cell><Table.Cell class="text-right tabular-nums">{formatCents(detail.expected_cents, $currencySymbol, $locale)}</Table.Cell></Table.Row>
            <Table.Row>
              <Table.Cell><strong>{$t.variance} ({varianceText})</strong></Table.Cell>
              <Table.Cell class="text-right tabular-nums">
                {#if detail.variance_cents > 0}
                  <Badge class="bg-success text-success-foreground hover:bg-success/90 tabular-nums"><strong>{formatSignedCents(detail.variance_cents, $currencySymbol, $locale)}</strong></Badge>
                {:else if detail.variance_cents < 0}
                  <Badge variant="destructive" class="tabular-nums"><strong>{formatSignedCents(detail.variance_cents, $currencySymbol, $locale)}</strong></Badge>
                {:else}
                  <Badge variant="secondary" class="tabular-nums"><strong>{formatSignedCents(detail.variance_cents, $currencySymbol, $locale)}</strong></Badge>
                {/if}
              </Table.Cell>
            </Table.Row>
          </Table.Body>
        </Table.Root>
        </div>
      </Card.Content>
    </Card.Root>
  </div>
{/if}

<AlertDialog.Root bind:open={confirmDelete}>
  <AlertDialog.Content
    onOpenAutoFocus={(e) => {
      e.preventDefault();
      cancelBtn?.focus();
    }}
  >
    <AlertDialog.Header>
      <AlertDialog.Title>{msg($t.deleteTitle, { id: `…${id.slice(-6)}` })}</AlertDialog.Title>
      <AlertDialog.Description>
        {$t.deleteBody}
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer>
      <AlertDialog.Cancel bind:ref={cancelBtn} disabled={deleting}>{$t.cancel}</AlertDialog.Cancel>
      <AlertDialog.Action disabled={deleting} onclick={remove}>
        <span class="grid items-center justify-items-center">
          <span class="col-start-1 row-start-1 {deleting ? 'invisible' : ''}">{$t.delete}</span>
          {#if deleting}
            <span class="col-start-1 row-start-1 flex items-center gap-2"><Spinner />{$t.deleting}</span>
          {/if}
        </span>
      </AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
