<script lang="ts">
  import { Input } from "$lib/components/ui/input";
  import { parseAmountToCents } from "./money";
  import { locale } from "./i18n";

  /**
   * Monetary input bound to integer cents.
   * - Shows the placeholder (not "0,00") until a value is entered, unless
   *   `hideZero` is false (used for editing existing stored values).
   * - Accepts digits with `.` or `,` decimals while typing.
   * - Invalid text never touches `cents`; it reverts on blur and reports
   *   via `onRevert` so callers can surface a message (the revert would
   *   otherwise silently discard what was typed).
   * - `invalid` is true only when non-empty text doesn't parse.
   * - Enter commits (blurs) and calls `onCommit`.
   */
  let {
    cents = $bindable(0),
    invalid = $bindable(false),
    id = undefined,
    placeholder = "",
    ariaLabel = undefined,
    describedBy = undefined,
    autofocus = false,
    hideZero = true,
    class: className = "",
    onCommit = undefined,
    onRevert = undefined,
  }: {
    cents?: number;
    invalid?: boolean;
    id?: string;
    placeholder?: string;
    ariaLabel?: string;
    describedBy?: string;
    autofocus?: boolean;
    hideZero?: boolean;
    class?: string;
    onCommit?: () => void;
    onRevert?: () => void;
  } = $props();

  let inputEl: HTMLInputElement | null = $state(null);
  let focused = $state(false);
  let lastEmitted = $state(cents);
  let text = $state(show(cents));

  function fmt(c: number): string {
    return new Intl.NumberFormat($locale === "fr" ? "fr-FR" : "en-US", {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(c / 100);
  }

  function show(c: number): string {
    if (hideZero && c === 0) return "";
    return fmt(c);
  }

  // $locale is referenced so a language switch reformats idle fields.
  $effect(() => {
    if (!focused && cents !== lastEmitted) {
      text = show(cents);
      lastEmitted = cents;
    } else if (!focused) {
      text = show(cents);
    }
  });

  $effect(() => {
    if (autofocus) inputEl?.focus();
  });

  function onInput(e: Event) {
    const el = e.target as HTMLInputElement;
    // Keep digits, decimal separators, spaces and currency symbols/codes
    // while typing; the parser decides what is valid.
    const cleaned = el.value.replace(/[^0-9.,\s$€£¥₹a-zA-Z]/g, "");
    // Collapse multiple separators to the first.
    const m = cleaned.match(/[.,]/g);
    if (m && m.length > 1) {
      const first = cleaned.search(/[.,]/);
      el.value =
        cleaned.slice(0, first + 1) + cleaned.slice(first + 1).replace(/[.,]/g, "");
    } else {
      el.value = cleaned;
    }
    text = el.value;
    const parsed = parseAmountToCents(text);
    if (parsed !== null) {
      cents = parsed;
      lastEmitted = parsed;
      invalid = false;
    } else {
      invalid = text.trim() !== "";
    }
  }

  function onBlur() {
    focused = false;
    // Non-empty text that never parsed is about to be discarded: let the
    // caller explain instead of silently reverting.
    if (text.trim() !== "" && parseAmountToCents(text) === null) {
      if (onRevert) onRevert();
      else invalid = false;
    } else {
      invalid = false;
    }
    text = show(cents);
    lastEmitted = cents;
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      inputEl?.blur();
      onCommit?.();
    } else if (e.key === "Escape") {
      e.preventDefault();
      text = show(cents);
      invalid = false;
      inputEl?.blur();
    }
  }
</script>

<Input
  {id}
  type="text"
  inputmode="decimal"
  {placeholder}
  aria-label={ariaLabel}
  aria-invalid={invalid ? true : undefined}
  aria-describedby={describedBy}
  value={text}
  bind:ref={inputEl}
  oninput={onInput}
  onfocus={(e) => {
    focused = true;
    (e.target as HTMLInputElement).select();
  }}
  onblur={onBlur}
  onkeydown={onKey}
  class="tabular-nums {className}"
/>
