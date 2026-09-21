<script lang="ts">
  import { onMount } from "svelte";
  import Router, { push, location } from "svelte-spa-router";
  import { Events, Clipboard } from "@wailsio/runtime";
  import { toast } from "svelte-sonner";
  import { ModeWatcher, mode, toggleMode } from "mode-watcher";
  import SunIcon from "@lucide/svelte/icons/sun";
  import MoonIcon from "@lucide/svelte/icons/moon";
  import LanguagesIcon from "@lucide/svelte/icons/languages";
  import { Button } from "$lib/components/ui/button";
  import { Separator } from "$lib/components/ui/separator";
  import * as Sidebar from "$lib/components/ui/sidebar";
  import { Toaster } from "$lib/components/ui/sonner";
  import { t, locale, setLocale, msg } from "$lib/i18n";
  import AppSidebar from "$lib/AppSidebar.svelte";
  import Count from "./routes/Count.svelte";
  import History from "./routes/History.svelte";
  import CountDetail from "./routes/CountDetail.svelte";
  import Settings from "./routes/Settings.svelte";
  import NotFound from "./routes/NotFound.svelte";

  const routes = {
    "/": Count,
    "/history": History,
    "/history/:id": CountDetail,
    "/settings": Settings,
    "*": NotFound,
  };

  const SIDEBAR_KEY = "cashcount-sidebar";

  function initialOpen(): boolean {
    try {
      const s = localStorage.getItem(SIDEBAR_KEY);
      if (s !== null) return s === "true";
    } catch {
      /* ignore */
    }
    return true;
  }

  let sidebarOpen = $state(initialOpen());
  let mainEl: HTMLElement | null = $state(null);

  function persistOpen(v: boolean): void {
    try {
      localStorage.setItem(SIDEBAR_KEY, String(v));
    } catch {
      /* ignore */
    }
  }

  function titleFor(path: string): string {
    if (path === "/") return $t.navCount;
    if (path === "/history" || path.startsWith("/history/")) return $t.navHistory;
    if (path === "/settings") return $t.navSettings;
    return $t.notFoundTitle;
  }

  function navigate(path: string, moveFocus: boolean) {
    push(path);
    document.title = `${titleFor(path)} — ${$t.appName}`;
    if (moveFocus) mainEl?.focus({ preventScroll: true });
  }

  onMount(() => {
    document.title = `${titleFor($location)} — ${$t.appName}`;    const offNavigate = Events.On("navigate", (ev) => {
      const path = (ev as { data?: unknown }).data;
      // Native-menu navigation: move focus into main since the menu
      // trigger itself is outside the page.
      if (typeof path === "string" && path.startsWith("/")) navigate(path, true);
    });
    const offExported = Events.On("export:history-done", (ev) => {
      const path = String((ev as { data?: unknown }).data ?? "");
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
    });
    return () => {
      offNavigate();
      offExported();
    };
  });

  // Keep the document title in sync with route + locale changes.
  $effect(() => {
    document.title = `${titleFor($location)} — ${$t.appName}`;
  });
</script>

<ModeWatcher track={false} defaultMode="dark" />
<Toaster position="top-right" richColors closeButton />

<a href="#main-content" class="skip-link">{$t.skipToContent}</a>
<Sidebar.Provider bind:open={sidebarOpen} onOpenChange={persistOpen}>
  <AppSidebar />
  <Sidebar.Inset>
    <header class="flex h-14 shrink-0 items-center gap-1 px-4">
      <Sidebar.Trigger aria-label={$t.toggleSidebar} />
      <Separator orientation="vertical" class="mx-1 h-5" />
      <span class="flex-1"></span>
      <Button
        variant="ghost"
        size="sm"
        onclick={() => setLocale($locale === "fr" ? "en" : "fr")}
        aria-label={msg($t.switchToLanguage, { lang: $locale === "fr" ? "English" : "Français" })}
        title={msg($t.switchToLanguage, { lang: $locale === "fr" ? "English" : "Français" })}
      >
        <LanguagesIcon data-icon="inline-start" aria-hidden="true" />
        {$locale === "fr" ? "EN" : "FR"}
      </Button>
      <Button variant="ghost" size="icon" onclick={toggleMode} aria-label={$t.toggleTheme}>
        {#if mode.current === "light"}
          <MoonIcon aria-hidden="true" />
        {:else}
          <SunIcon aria-hidden="true" />
        {/if}
      </Button>
    </header>
    <main id="main-content" bind:this={mainEl} tabindex="-1" class="outline-none mx-auto w-full max-w-5xl flex-1 px-4 py-6 pb-16">
      <Router {routes} />
    </main>
  </Sidebar.Inset>
</Sidebar.Provider>
