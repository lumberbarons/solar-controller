import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

import {
  BATTERY_PROFILE_FIELDS,
  CHARGING_PARAMETER_FIELDS,
  EDITABLE_DURATION_FIELDS,
  EDITABLE_VOLTAGE_FIELDS,
  INFO_FIELDS,
  METRIC_FIELDS,
  TIME_FIELDS,
  VOLTGO_BATTERY_REF_FIELDS,
  VOLTGO_CELL_FIELDS,
  VOLTGO_INDEX_FIELDS,
  VOLTGO_INFO_FIELDS,
  VOLTGO_METRIC_FIELDS,
} from './types';

/**
 * docs/api-contract.json is the shared record of what each endpoint emits. The
 * Go handler tests assert the handlers produce exactly these keys; these tests
 * assert the frontend declares exactly the same ones. A field renamed on either
 * side fails one of the two suites until both agree.
 */
// Located by walking up from the working directory rather than from
// import.meta.url, which the jsdom environment reports as an http:// URL that
// readFileSync rejects. Walking up keeps this working whether vitest is invoked
// from site/ or from the repository root.
function findContractPath(): string {
  let dir = process.cwd();
  for (let depth = 0; depth < 5; depth++) {
    const candidate = resolve(dir, 'docs', 'api-contract.json');
    if (existsSync(candidate)) {
      return candidate;
    }
    const parent = dirname(dir);
    if (parent === dir) {
      break;
    }
    dir = parent;
  }
  throw new Error(`could not find docs/api-contract.json above ${process.cwd()}`);
}

const contract = JSON.parse(readFileSync(findContractPath(), 'utf8')) as {
  endpoints: Record<string, string[]>;
};

function contractFields(endpoint: string): string[] {
  const fields = contract.endpoints[endpoint];
  if (!fields) {
    throw new Error(`docs/api-contract.json has no entry for "${endpoint}"`);
  }
  return fields;
}

describe('API contract', () => {
  it.each([
    ['GET /api/info', INFO_FIELDS],
    ['GET /api/epever/metrics', METRIC_FIELDS],
    ['GET /api/epever/battery-profile', BATTERY_PROFILE_FIELDS],
    ['GET /api/epever/charging-parameters', CHARGING_PARAMETER_FIELDS],
    ['GET /api/epever/time', TIME_FIELDS],
    ['GET /api/voltgo', VOLTGO_INDEX_FIELDS],
    ['GET /api/voltgo#batteries[]', VOLTGO_BATTERY_REF_FIELDS],
    ['GET /api/voltgo/{id}/metrics', VOLTGO_METRIC_FIELDS],
    ['GET /api/voltgo/{id}/metrics#cells[]', VOLTGO_CELL_FIELDS],
    ['GET /api/voltgo/{id}/info', VOLTGO_INFO_FIELDS],
  ])('%s declares the fields the Go handler returns', (endpoint, declared) => {
    expect([...declared].sort()).toEqual(contractFields(endpoint));
  });
});

describe('editable charging parameter fields', () => {
  it('are all real charging parameters', () => {
    const known = new Set<string>(CHARGING_PARAMETER_FIELDS);
    for (const field of [...EDITABLE_DURATION_FIELDS, ...EDITABLE_VOLTAGE_FIELDS]) {
      expect(known.has(field)).toBe(true);
    }
  });

  it('do not classify a field as both a duration and a voltage', () => {
    const durations = new Set<string>(EDITABLE_DURATION_FIELDS);
    const overlap = EDITABLE_VOLTAGE_FIELDS.filter(field => durations.has(field));
    expect(overlap).toEqual([]);
  });
});
