/**
 * Types for every payload crossing the /api boundary.
 *
 * This is the single place to update when a Go handler changes shape. The field
 * arrays are exported as runtime values, not just types, so `types.test.ts` can
 * compare them against `docs/api-contract.json` — which the Go handler tests
 * verify from the other side. A field renamed in Go without a matching change
 * here fails CI rather than silently rendering `undefined` in the browser.
 */

/** GET /api/info */
export const INFO_FIELDS = ['buildTime', 'gitCommit', 'version'] as const;

export type VersionInfo = {
  [K in (typeof INFO_FIELDS)[number]]: string;
};

/** GET /api/epever/metrics */
export const METRIC_FIELDS = [
  'arrayCurrent',
  'arrayPower',
  'arrayVoltage',
  'batterySoc',
  'batteryTemp',
  'batteryVoltage',
  'chargingCurrent',
  'chargingPower',
  'chargingStatus',
  'collectionTime',
  'deviceTemp',
  'energyGeneratedDaily',
  'timestamp',
] as const;

export type MetricField = (typeof METRIC_FIELDS)[number];

export type EpeverMetrics = {
  [K in MetricField]: number;
};

/**
 * Charging status codes from the `chargingStatus` metric. The device reports a
 * raw code; anything outside this set renders as "Unknown".
 */
export const CHARGING_STATUS_LABELS: Record<number, string> = {
  0: 'Not charging',
  1: 'Float',
  2: 'Boost',
  3: 'Equalization',
};

/** GET and PATCH /api/epever/battery-profile */
export const BATTERY_PROFILE_FIELDS = [
  'batteryCapacity',
  'batteryType',
  'tempCompCoefficient',
] as const;

export type BatteryProfileField = (typeof BATTERY_PROFILE_FIELDS)[number];

/** The battery types the device accepts; `unknown` is read-only. */
export const BATTERY_TYPES = ['sealed', 'gel', 'flooded', 'userDefined'] as const;

export type BatteryType = (typeof BATTERY_TYPES)[number] | 'unknown';

export type BatteryProfile = {
  batteryType: BatteryType;
  batteryCapacity: number;
  tempCompCoefficient: number;
};

/**
 * A PATCH body sends only the fields that changed. `batteryType` is a string
 * enum; the numeric fields are parsed from text inputs before sending.
 */
export type BatteryProfilePatch = {
  batteryType?: BatteryType;
  batteryCapacity?: number;
  tempCompCoefficient?: number;
};

/** GET and PATCH /api/epever/charging-parameters */
export const CHARGING_PARAMETER_FIELDS = [
  'batteryTempLowerLimit',
  'batteryTempUpperLimit',
  'boostDuration',
  'boostReconnectChargingVoltage',
  'boostVoltage',
  'chargingLimitVoltage',
  'controllerTempLowerLimit',
  'controllerTempUpperLimit',
  'dischargingLimitVoltage',
  'equalizationCycle',
  'equalizationDuration',
  'equalizationVoltage',
  'floatVoltage',
  'lowVoltDisconnectVoltage',
  'lowVoltReconnectVoltage',
  'overVoltDisconnectVoltage',
  'overVoltReconnectVoltage',
  'underVoltWarningReconnectVoltage',
  'underVoltWarningVoltage',
] as const;

export type ChargingParameterField = (typeof CHARGING_PARAMETER_FIELDS)[number];

export type ChargingParameters = {
  [K in ChargingParameterField]: number;
};

export type ChargingParametersPatch = Partial<ChargingParameters>;

/**
 * The subset of charging parameters the config screen exposes as inputs, split
 * by whether the device stores them as whole minutes/days or as a voltage. The
 * split drives parseInt versus parseFloat when building a PATCH body, so a
 * duration is never sent as a fraction.
 */
export const EDITABLE_DURATION_FIELDS = [
  'boostDuration',
  'equalizationCycle',
  'equalizationDuration',
] as const;

export const EDITABLE_VOLTAGE_FIELDS = [
  'boostVoltage',
  'boostReconnectChargingVoltage',
  'chargingLimitVoltage',
  'dischargingLimitVoltage',
  'equalizationVoltage',
  'floatVoltage',
  'lowVoltDisconnectVoltage',
  'lowVoltReconnectVoltage',
  'overVoltDisconnectVoltage',
  'overVoltReconnectVoltage',
  'underVoltWarningReconnectVoltage',
  'underVoltWarningVoltage',
] as const;

/** GET and PATCH /api/epever/time */
export const TIME_FIELDS = ['time'] as const;

export type ControllerTime = {
  /** RFC 3339, as produced by Go's time.Time marshalling. */
  time: string;
};

/**
 * Form state holds raw input strings alongside fetched numbers, because a text
 * input yields a string until it is parsed on submit.
 */
export type FormValue = number | string;

export type BatteryProfileForm = {
  batteryType: BatteryType | '';
  batteryCapacity: FormValue;
  tempCompCoefficient: FormValue;
};

export type ChargingParametersForm = {
  [K in ChargingParameterField]: FormValue;
};

/** The shape Gin uses for handler error responses: `{"error": "..."}`. */
export type ApiError = {
  error?: string;
};
