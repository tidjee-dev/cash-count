/** Backend access layer over the generated Wails bindings. */

import { settings, denominations } from "./state";
import { SettingsSvc, DenominationSvc } from "./services";
import type { Denomination, Settings } from "./state";

export interface AppData {
  settings: Settings;
  denoms: Denomination[];
}

/** Load settings + active denominations and refresh the shared stores. */
export async function loadAppData(): Promise<AppData> {
  const [s, list] = await Promise.all([
    SettingsSvc.GetSettings() as Promise<Settings>,
    DenominationSvc.List(true) as Promise<Denomination[]>,
  ]);
  settings.set(s);
  denominations.set(list);
  return { settings: s, denoms: list };
}
