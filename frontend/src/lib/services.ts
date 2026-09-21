/** Stable frontend façade over the Wails-generated bindings.
 *  Generated files live per Go package; screens import these namespaces so only
 *  this file changes if binding layout changes. Run
 *  `wails3 generate bindings` after backend changes. */
import * as SettingsSvc from "../../bindings/github.com/tidjee-dev/cash-count/backend/settings/service.js";
import * as DenominationSvc from "../../bindings/github.com/tidjee-dev/cash-count/backend/denominations/service.js";
import * as CountSvc from "../../bindings/github.com/tidjee-dev/cash-count/backend/counts/service.js";

export { SettingsSvc, DenominationSvc, CountSvc };
