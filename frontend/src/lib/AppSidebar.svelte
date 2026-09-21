<script lang="ts">
  import { link, location } from "svelte-spa-router";
  import CalculatorIcon from "@lucide/svelte/icons/calculator";
  import HistoryIcon from "@lucide/svelte/icons/history";
  import SettingsIcon from "@lucide/svelte/icons/settings";
  import WalletIcon from "@lucide/svelte/icons/wallet";
  import StoreIcon from "@lucide/svelte/icons/store";
  import * as Sidebar from "$lib/components/ui/sidebar";
  import { settings } from "$lib/state";
  import { loadAppData } from "$lib/backend";
  import { t } from "$lib/i18n";

  const items = [
    {
      href: "#/",
      path: "/",
      icon: CalculatorIcon,
      get label() {
        return $t.navCount;
      },
    },
    {
      href: "#/history",
      path: "/history",
      icon: HistoryIcon,
      get label() {
        return $t.navHistory;
      },
    },
    {
      href: "#/settings",
      path: "/settings",
      icon: SettingsIcon,
      get label() {
        return $t.navSettings;
      },
    },
  ];

  $effect(() => {
    loadAppData().catch(() => {});
  });

  function isActive(item: string): boolean {
    if (item === "/") return $location === "/";
    return $location === item || $location.startsWith(item + "/");
  }
</script>

<Sidebar.Root collapsible="icon" variant="inset">
  <Sidebar.Header>
    <Sidebar.Menu>
      <Sidebar.MenuItem>
        <Sidebar.MenuButton size="lg" tooltipContent={$t.appName}>
          {#snippet child({ props })}
            <a href="#/" use:link {...props} aria-label={$t.appName}>
              <span
                class="bg-primary text-primary-foreground flex size-8 items-center justify-center rounded-lg"
              >
                <WalletIcon />
              </span>
              <span class="text-base font-bold tracking-tight group-data-[collapsible=icon]:hidden"
                >{$t.appName}</span
              >
            </a>
          {/snippet}
        </Sidebar.MenuButton>
      </Sidebar.MenuItem>
    </Sidebar.Menu>
  </Sidebar.Header>
  <Sidebar.Content>
    <Sidebar.Group>
      <Sidebar.GroupLabel>{$t.navGroup}</Sidebar.GroupLabel>
      <Sidebar.GroupContent>
        <Sidebar.Menu>
          {#each items as item (item.path)}
            {@const Icon = item.icon}
            {@const active = isActive(item.path)}
            <Sidebar.MenuItem>
              <Sidebar.MenuButton isActive={active} tooltipContent={item.label}>
                {#snippet child({ props })}
                  <a
                    href={item.href}
                    use:link
                    {...props}
                    aria-current={active ? "page" : undefined}
                  >
                    <Icon />
                    <span>{item.label}</span>
                  </a>
                {/snippet}
              </Sidebar.MenuButton>
            </Sidebar.MenuItem>
          {/each}
        </Sidebar.Menu>
      </Sidebar.GroupContent>
    </Sidebar.Group>
  </Sidebar.Content>
  <Sidebar.Footer>
    <Sidebar.Menu>
      <Sidebar.MenuItem>
        <div
          class="text-sidebar-foreground flex w-full items-center gap-2 overflow-hidden rounded-md p-2 text-sm"
          title={$settings.store_name || $t.storeCurrency}
        >
          <StoreIcon class="size-4 shrink-0" />
          <span
            class="min-w-0 flex-1 truncate font-medium group-data-[collapsible=icon]:hidden"
          >
            {$settings.store_name || "—"}
          </span>
          <span
            class="text-muted-foreground shrink-0 text-xs tabular-nums group-data-[collapsible=icon]:hidden"
          >
            {$settings.currency_symbol}
          </span>
        </div>
      </Sidebar.MenuItem>
    </Sidebar.Menu>
  </Sidebar.Footer>
  <Sidebar.Rail />
</Sidebar.Root>
