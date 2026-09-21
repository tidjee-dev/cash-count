/** Shared frontend types mirroring the Go service models. */

export interface Denomination {
  id: string;
  label: string;
  value_cents: number;
  kind: string;
  sort_order: number;
  active: boolean;
}

export interface DenomState extends Denomination {
  qty: number;
}

export interface DropDraft {
  amount_cents: number;
  note: string;
}

export interface CountSummary {
  id: string;
  created_at: string;
  type: string;
  expected_cents: number;
  counted_cents: number;
  drops_cents: number;
  variance_cents: number;
}

export interface CountItemDetail {
  id: string;
  denomination_id: string;
  denomination_label: string;
  value_cents: number;
  quantity: number;
  subtotal_cents: number;
}

export interface CountDropDetail {
  id: string;
  amount_cents: number;
  note: string;
  created_at: string;
}

export interface CountDetail extends CountSummary {
  note: string;
  items: CountItemDetail[];
  drops: CountDropDetail[];
}
