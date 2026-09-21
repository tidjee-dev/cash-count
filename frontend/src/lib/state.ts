import { writable } from "svelte/store";

export interface Denomination {
  id: string;
  label: string;
  value_cents: number;
  kind: string;
  sort_order: number;
  active: boolean;
}

export interface Settings {
  id: number;
  store_name: string;
  currency_label: string;
  currency_symbol: string;
}

/** Cached settings (currency symbol drives all money formatting). */
export const settings = writable<Settings>({
  id: 0,
  store_name: "",
  currency_label: "EUR",
  currency_symbol: "€",
});

/** Cached active denominations. */
export const denominations = writable<Denomination[]>([]);

/** Bump to force list screens to reload after a mutation. */
export const dataVersion = writable(0);

export function notifyDataChanged(): void {
  dataVersion.update((v) => v + 1);
}
