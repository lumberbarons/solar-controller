import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi, type Mock } from 'vitest';

import Config from './config';
import type { BatteryProfile, ChargingParameters } from '../api/types';

// axios is mocked rather than intercepted at the network layer so each test can
// assert the exact PATCH body the panel sends. isAxiosError is reimplemented
// because the component uses it to decide how to describe a failure.
vi.mock('axios', () => {
  const isAxiosError = (error: unknown): boolean =>
    Boolean((error as { isAxiosError?: boolean } | null)?.isAxiosError);
  return {
    default: { get: vi.fn(), patch: vi.fn(), isAxiosError },
  };
});

import axios from 'axios';

const mockedGet = (axios as unknown as { get: Mock }).get;
const mockedPatch = (axios as unknown as { patch: Mock }).patch;

const BATTERY_PROFILE: BatteryProfile = {
  batteryType: 'userDefined',
  batteryCapacity: 100,
  tempCompCoefficient: 3,
};

const CHARGING_PARAMETERS: ChargingParameters = {
  boostDuration: 120,
  equalizationCycle: 30,
  equalizationDuration: 120,
  boostVoltage: 14.6,
  boostReconnectChargingVoltage: 13.2,
  floatVoltage: 13.8,
  equalizationVoltage: 14.6,
  chargingLimitVoltage: 15.0,
  overVoltDisconnectVoltage: 16.0,
  overVoltReconnectVoltage: 15.0,
  lowVoltDisconnectVoltage: 11.1,
  lowVoltReconnectVoltage: 12.6,
  underVoltWarningVoltage: 12.0,
  underVoltWarningReconnectVoltage: 12.2,
  dischargingLimitVoltage: 10.8,
  batteryTempUpperLimit: 45,
  batteryTempLowerLimit: -45,
  controllerTempUpperLimit: 45,
  controllerTempLowerLimit: 40,
};

/**
 * An error shaped the way axios rejects: a real Error subclass carrying
 * isAxiosError and, for a server response, a response object. It has to be an
 * Error instance because the component falls back to `error.message` only for
 * genuine Errors — a plain object would stringify to "[object Object]".
 */
function axiosError(options: {
  status?: number;
  statusText?: string;
  backendError?: string;
  message?: string;
}) {
  const error = new Error(options.message ?? 'Request failed') as Error & {
    isAxiosError: boolean;
    response?: { status: number; statusText: string; data: { error?: string } };
  };
  error.isAxiosError = true;
  if (options.status !== undefined) {
    error.response = {
      status: options.status,
      statusText: options.statusText ?? '',
      data: options.backendError ? { error: options.backendError } : {},
    };
  }
  return error;
}

function resolveLoad(profile: BatteryProfile = BATTERY_PROFILE) {
  mockedGet.mockImplementation((url: string) => {
    if (url === '/api/epever/battery-profile') {
      return Promise.resolve({ data: structuredClone(profile) });
    }
    if (url === '/api/epever/charging-parameters') {
      return Promise.resolve({ data: structuredClone(CHARGING_PARAMETERS) });
    }
    return Promise.reject(new Error(`unexpected GET ${url}`));
  });
}

/** Renders the panel and waits for the initial fetch to populate the inputs. */
async function renderLoaded(profile: BatteryProfile = BATTERY_PROFILE) {
  resolveLoad(profile);
  render(<Config />);
  await waitFor(() => {
    expect(screen.getByLabelText(/Battery Capacity/)).toHaveValue('100');
  });
}

function setInput(label: RegExp, value: string) {
  fireEvent.change(screen.getByLabelText(label), { target: { value } });
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('battery profile panel', () => {
  it('renders the fetched values', async () => {
    await renderLoaded();

    expect(screen.getByLabelText(/Battery Capacity/)).toHaveValue('100');
    expect(screen.getByLabelText(/Temp Comp Coeff/)).toHaveValue('3');
  });

  it('sends only the changed fields, parsed to numbers', async () => {
    await renderLoaded();
    mockedPatch.mockResolvedValue({ data: structuredClone(BATTERY_PROFILE) });

    setInput(/Battery Capacity/, '200');
    fireEvent.click(screen.getByRole('button', { name: 'Save Battery Profile' }));

    await waitFor(() => expect(mockedPatch).toHaveBeenCalled());
    expect(mockedPatch).toHaveBeenCalledWith('/api/epever/battery-profile', {
      batteryCapacity: 200,
    });
  });

  it('parses the temperature compensation coefficient as a decimal', async () => {
    await renderLoaded();
    mockedPatch.mockResolvedValue({ data: structuredClone(BATTERY_PROFILE) });

    setInput(/Temp Comp Coeff/, '3.5');
    fireEvent.click(screen.getByRole('button', { name: 'Save Battery Profile' }));

    await waitFor(() => expect(mockedPatch).toHaveBeenCalled());
    expect(mockedPatch).toHaveBeenCalledWith('/api/epever/battery-profile', {
      tempCompCoefficient: 3.5,
    });
  });

  it('sends an empty body when nothing changed, so no register is rewritten', async () => {
    await renderLoaded();
    mockedPatch.mockResolvedValue({ data: structuredClone(BATTERY_PROFILE) });

    fireEvent.click(screen.getByRole('button', { name: 'Save Battery Profile' }));

    await waitFor(() => expect(mockedPatch).toHaveBeenCalled());
    expect(mockedPatch).toHaveBeenCalledWith('/api/epever/battery-profile', {});
  });

  it('surfaces the backend error message when the save is rejected', async () => {
    await renderLoaded();
    mockedPatch.mockRejectedValue(
      axiosError({ status: 400, backendError: 'batteryCapacity (5000) out of range [1, 3000] Ah' }),
    );

    setInput(/Battery Capacity/, '5000');
    fireEvent.click(screen.getByRole('button', { name: 'Save Battery Profile' }));

    expect(
      await screen.findByText(
        'Failed to save battery profile: batteryCapacity (5000) out of range [1, 3000] Ah',
      ),
    ).toBeInTheDocument();
  });

  it('reports the HTTP status when the backend sends no error body', async () => {
    await renderLoaded();
    mockedPatch.mockRejectedValue(axiosError({ status: 500, statusText: 'Internal Server Error' }));

    setInput(/Battery Capacity/, '200');
    fireEvent.click(screen.getByRole('button', { name: 'Save Battery Profile' }));

    expect(
      await screen.findByText('Failed to save battery profile: 500 Internal Server Error'),
    ).toBeInTheDocument();
  });

  it('reports a transport failure that has no response at all', async () => {
    await renderLoaded();
    mockedPatch.mockRejectedValue(axiosError({ message: 'Network Error' }));

    setInput(/Battery Capacity/, '200');
    fireEvent.click(screen.getByRole('button', { name: 'Save Battery Profile' }));

    expect(
      await screen.findByText('Failed to save battery profile: Network Error'),
    ).toBeInTheDocument();
  });
});

describe('charging parameters panel', () => {
  it('sends a changed voltage as a decimal', async () => {
    await renderLoaded();
    mockedPatch.mockResolvedValue({ data: structuredClone(CHARGING_PARAMETERS) });

    setInput(/Boost Voltage/, '14.8');
    fireEvent.click(screen.getByRole('button', { name: 'Save Charging Parameters' }));

    await waitFor(() => expect(mockedPatch).toHaveBeenCalled());
    expect(mockedPatch).toHaveBeenCalledWith('/api/epever/charging-parameters', {
      boostVoltage: 14.8,
    });
  });

  it('sends a changed duration as a whole number', async () => {
    await renderLoaded();
    mockedPatch.mockResolvedValue({ data: structuredClone(CHARGING_PARAMETERS) });

    setInput(/Boost Duration/, '90.7');
    fireEvent.click(screen.getByRole('button', { name: 'Save Charging Parameters' }));

    await waitFor(() => expect(mockedPatch).toHaveBeenCalled());
    expect(mockedPatch).toHaveBeenCalledWith('/api/epever/charging-parameters', {
      boostDuration: 90,
    });
  });

  it('sends every changed field together and nothing else', async () => {
    await renderLoaded();
    mockedPatch.mockResolvedValue({ data: structuredClone(CHARGING_PARAMETERS) });

    setInput(/Float Voltage/, '13.5');
    setInput(/Equalization Cycle/, '45');
    setInput(/Discharging Limit Voltage/, '10.5');
    fireEvent.click(screen.getByRole('button', { name: 'Save Charging Parameters' }));

    await waitFor(() => expect(mockedPatch).toHaveBeenCalled());
    expect(mockedPatch).toHaveBeenCalledWith('/api/epever/charging-parameters', {
      floatVoltage: 13.5,
      equalizationCycle: 45,
      dischargingLimitVoltage: 10.5,
    });
  });

  it('never sends the read-only temperature limits', async () => {
    await renderLoaded();
    mockedPatch.mockResolvedValue({ data: structuredClone(CHARGING_PARAMETERS) });

    setInput(/Boost Voltage/, '14.8');
    fireEvent.click(screen.getByRole('button', { name: 'Save Charging Parameters' }));

    await waitFor(() => expect(mockedPatch).toHaveBeenCalled());
    const [, body] = mockedPatch.mock.calls[0] as [string, Record<string, unknown>];
    expect(Object.keys(body)).not.toContain('batteryTempUpperLimit');
    expect(Object.keys(body)).not.toContain('controllerTempUpperLimit');
  });

  it('surfaces the backend error message when the save is rejected', async () => {
    await renderLoaded();
    mockedPatch.mockRejectedValue(
      axiosError({ status: 400, backendError: 'boostVoltage (700.00) out of range' }),
    );

    setInput(/Boost Voltage/, '700');
    fireEvent.click(screen.getByRole('button', { name: 'Save Charging Parameters' }));

    expect(
      await screen.findByText('Failed to save charging parameters: boostVoltage (700.00) out of range'),
    ).toBeInTheDocument();
  });

  it('is not editable unless the saved battery type is userDefined', async () => {
    await renderLoaded({ ...BATTERY_PROFILE, batteryType: 'sealed' });

    expect(screen.getByLabelText(/Boost Voltage/)).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Save Charging Parameters' })).toBeDisabled();
    expect(
      screen.getByText("Set Battery Type to 'User Defined' and save to edit charging parameters"),
    ).toBeInTheDocument();
  });
});

describe('initial load', () => {
  it('shows an error and no values when the fetch fails', async () => {
    mockedGet.mockRejectedValue(axiosError({ status: 503, statusText: 'Service Unavailable' }));

    render(<Config />);

    expect(
      await screen.findByText('Failed to load configuration: 503 Service Unavailable'),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Save Battery Profile' })).toBeDisabled();
  });
});
