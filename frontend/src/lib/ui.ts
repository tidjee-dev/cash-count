import { derived } from "svelte/store";
import { settings } from "./state";

/** Reactive currency symbol for all money formatting. */
export const currencySymbol = derived(settings, ($s) => $s.currency_symbol || "€");
