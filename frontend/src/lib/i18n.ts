import { writable, derived } from "svelte/store";
import en from "./locales/en.json";
import fr from "./locales/fr.json";
export type Locale = "fr" | "en";
export type Dict = typeof en;

// en is the reference: a missing key in fr fails here.
const dicts: Record<Locale, Dict> = {
	en,
	fr
};
const KEY = "cashcount-locale";
function initial(): Locale {
	try {
		const s = localStorage.getItem(KEY);
		if (s === "en" || s === "fr") return s;
	} catch {
		/* ignore */
	}
	return "fr";
}
export const locale = writable<Locale>(initial());
if (typeof document !== "undefined") {
	// Apply the stored locale before first render so lang is correct
	// even before any setLocale() call (e.g. screen readers, spellcheck).
	let current: Locale = "fr";
	const unsub = locale.subscribe((l) => (current = l));
	unsub();
	document.documentElement.lang = current;
}
export function setLocale(l: Locale): void {
	locale.set(l);
	try {
		localStorage.setItem(KEY, l);
	} catch {
		/* ignore */
	}
	if (typeof document !== "undefined") document.documentElement.lang = l;
}

/** Synchronous read for non-reactive contexts (formatters). */
export function getLocaleSnapshot(): Locale {
	let v: Locale = "fr";
	const unsub = locale.subscribe($l => v = $l);
	unsub();
	return v;
}

/** Reactive dictionary for the current locale. Usage: {$t.saveCount} */
export const t = derived(locale, ($l): Dict => dicts[$l]);

/** Fill {placeholders} in a template (all occurrences). */
export function msg(template: string, vars: Record<string, string | number>): string {
	let out = template;
	for (const [k, v] of Object.entries(vars)) out = out.split(`{${k}}`).join(String(v));
	return out;
}
