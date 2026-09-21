/** Money helpers: integer cents everywhere, formatting/parsing in one place. */

import { getLocaleSnapshot, type Locale } from "./i18n";
function loc(l?: Locale): Locale {
	return l ?? getLocaleSnapshot();
}

/** Format integer cents, e.g. -450 -> "-$4.50" (en) / "-4,50 $US" style handled per locale. */
export function formatCents(cents: number, symbol = "€", locale?: Locale): string {
	const l = loc(locale);
	const sign = cents < 0 ? "-" : "";
	const abs = Math.abs(Math.trunc(cents));
	const whole = Math.floor(abs / 100);
	const frac = abs % 100;
	if (l === "fr") {
		const grouped = whole.toString().replace(/\B(?=(\d{3})+(?!\d))/g, " ");
		return `${sign}${grouped},${frac.toString().padStart(2, "0")} ${symbol}`;
	}
	const grouped = whole.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
	return `${sign}${symbol}${grouped}.${frac.toString().padStart(2, "0")}`;
}

/** Format with an explicit + sign for positive values (variance display). */
export function formatSignedCents(cents: number, symbol = "€", locale?: Locale): string {
	if (cents > 0) return `+${formatCents(cents, symbol, locale)}`;
	return formatCents(cents, symbol, locale);
}

/**
 * Parse a user-entered amount ("12.34", "12,34", "1 234,56 €", "$1,234.56")
 * into integer cents. Accepts both dot and comma decimals; when both are
 * present the last one wins, and a lone 3-digit group is read as thousands.
 * Common currency symbols/codes are ignored. Returns null when invalid.
 */
export function parseAmountToCents(raw: string): number | null {
  let s = raw
    .trim()
    .replace(/\s|\u00a0|\u202f/g, "")
    .replace(/€|£|\$|¥|₹/g, "")
    .replace(/EUR|USD|GBP|CHF|CAD|AUD|JPY|CNY/gi, "");
  if (s === "" || !/^[0-9.,]+$/.test(s)) return null;
  const lastDot = s.lastIndexOf(".");
  const lastComma = s.lastIndexOf(",");
  let whole = s;
  let frac = "";
  if (lastDot >= 0 || lastComma >= 0) {
    const dec = Math.max(lastDot, lastComma);
    whole = s.slice(0, dec);
    frac = s.slice(dec + 1);
    // Lone separator with exactly 3 trailing digits means thousands.
    if (lastDot < 0 !== lastComma < 0 && frac.length === 3) {
      whole = s;
      frac = "";
    }
  }
  whole = whole.replace(/[.,]/g, "");
  frac = frac.replace(/[.,]/g, "");
  if (!/^\d+$/.test(whole)) return null;
  if (frac !== "" && !/^\d{1,2}$/.test(frac)) return null;
  const cents = Number(whole) * 100 + Number((frac + "00").slice(0, 2));
  if (!Number.isSafeInteger(cents) || cents < 0) return null;
  return cents;
}

/** Variance state label for color-coding. */
export function varianceState(varianceCents: number): "over" | "short" | "balanced" {
	if (varianceCents > 0) return "over";
	if (varianceCents < 0) return "short";
	return "balanced";
}
