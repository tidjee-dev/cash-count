/** Shared display formatting. */

import { getLocaleSnapshot, type Dict, type Locale } from "./i18n";

/** Localized display label for a count type ("Unknown" falls through). */
export function countTypeLabel(v: string, d: Dict): string {
  if (v === "shift_open") return d.typeOpen;
  if (v === "shift_close") return d.typeClose;
  if (v === "audit") return d.typeAudit;
  if (v === "donation_urne") return d.typeDonationUrne;
  return d.unknownType;
}

/** Format an ISO/RFC3339 timestamp for UI (compact date + time). Falls back to "–". */
export function fmtDate(iso: string, locale?: Locale): string {
  const l = locale ?? getLocaleSnapshot();
  try {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return "–";
    return d.toLocaleString(l === "fr" ? "fr-FR" : "en-US", {
      dateStyle: "medium",
      timeStyle: "short",
    });
  } catch {
    return "–";
  }
}
