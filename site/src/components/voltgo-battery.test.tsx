import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi, type Mock } from 'vitest';

import VoltgoBatteryBank from './voltgo-battery';
import type { VoltgoBatteryRef, VoltgoInfo, VoltgoMetrics } from '../api/types';

vi.mock('axios', () => {
  const isAxiosError = (error: unknown): boolean =>
    Boolean((error as { isAxiosError?: boolean } | null)?.isAxiosError);
  return {
    default: { get: vi.fn(), isAxiosError },
  };
});

import axios from 'axios';

const mockedGet = (axios as unknown as { get: Mock }).get;

const METRICS: VoltgoMetrics = {
  timestamp: 1699000000,
  collectionTime: 1.8,
  voltage: 13.31,
  current: 12.4,
  soc: 74,
  soh: 99,
  temperature: 19.5,
  temperatures: [19, 20],
  cellCount: 4,
  cells: [
    { index: 0, voltage: 3.331 },
    { index: 1, voltage: 3.325 },
    { index: 2, voltage: 3.338 },
    { index: 3, voltage: 3.326 },
  ],
};

const INFO: VoltgoInfo = {
  chemistry: 'LiFePO4',
  nominalVoltage: 12.8,
  capacityAh: 100,
  deviceStrings: ['VG-100'],
};

function httpError(status: number, statusText: string) {
  const error = new Error(`Request failed with status code ${status}`) as Error & {
    isAxiosError: boolean;
    response: { status: number; statusText: string };
  };
  error.isAxiosError = true;
  error.response = { status, statusText };
  return error;
}

/** Answers each voltgo endpoint; anything not listed 404s, as an unregistered route does. */
function mockApi(responses: Record<string, { status: number; data?: unknown }>) {
  mockedGet.mockImplementation((url: string) => {
    const response = responses[url];
    if (!response) {
      return Promise.reject(httpError(404, 'Not Found'));
    }
    return Promise.resolve({ status: response.status, data: response.data ?? '' });
  });
}

const BANK_A: VoltgoBatteryRef = { id: 'bank-a', address: 'A4:C1:37:43:A4:33' };
const BANK_B: VoltgoBatteryRef = { id: 'bank-b', address: 'A4:C1:37:43:A4:42' };

/** The index answer for a given set of batteries. */
const index = (batteries: VoltgoBatteryRef[]) => ({
  '/api/voltgo': { status: 200, data: { batteries } },
});

/** One battery, reporting normally: the single-pack deployment. */
const withMetrics = (metrics: VoltgoMetrics = METRICS) =>
  mockApi({
    ...index([BANK_A]),
    '/api/voltgo/bank-a/metrics': { status: 200, data: metrics },
    '/api/voltgo/bank-a/info': { status: 200, data: INFO },
  });

beforeEach(() => {
  vi.clearAllMocks();
});

describe('voltgo battery panel', () => {
  it('renders the pack metrics', async () => {
    withMetrics();

    render(<VoltgoBatteryBank refreshKey={0} />);

    await waitFor(() => expect(screen.getByText('74 %')).toBeInTheDocument());
    expect(screen.getByText('13.31 V')).toBeInTheDocument();
    expect(screen.getByText('99 %')).toBeInTheDocument();
    expect(screen.getByText('19.5 °C')).toBeInTheDocument();
  });

  it('shows the battery description once the info endpoint answers', async () => {
    withMetrics();

    render(<VoltgoBatteryBank refreshKey={0} />);

    await waitFor(() =>
      expect(screen.getByText('LiFePO4 · 12.8 V · 100 Ah')).toBeInTheDocument(),
    );
  });

  it('still renders the pack when the info endpoint has no data yet', async () => {
    mockApi({
      ...index([BANK_A]),
      '/api/voltgo/bank-a/metrics': { status: 200, data: METRICS },
      '/api/voltgo/bank-a/info': { status: 204 },
    });

    render(<VoltgoBatteryBank refreshKey={0} />);

    await waitFor(() => expect(screen.getByText('74 %')).toBeInTheDocument());
    expect(screen.queryByText(/LiFePO4/)).not.toBeInTheDocument();
  });

  it.each([
    [12.4, '+12.40 A', 'Charging'],
    [-8.75, '-8.75 A', 'Discharging'],
    [0.01, '+0.01 A', 'Idle'],
    [-0.01, '-0.01 A', 'Idle'],
  ])('renders current %p as %s (%s)', async (current, displayed, flow) => {
    withMetrics({ ...METRICS, current });

    render(<VoltgoBatteryBank refreshKey={0} />);

    await waitFor(() => expect(screen.getByText(displayed)).toBeInTheDocument());
    expect(screen.getByText(flow)).toBeInTheDocument();
  });

  it('highlights the highest and lowest cell and reports the spread', async () => {
    withMetrics();

    render(<VoltgoBatteryBank refreshKey={0} />);

    await waitFor(() =>
      expect(screen.getByLabelText('Cell 2 · 3.338 V (highest)')).toBeInTheDocument(),
    );
    expect(screen.getByLabelText('Cell 1 · 3.325 V (lowest)')).toBeInTheDocument();
    expect(screen.getByLabelText('Cell 0 · 3.331 V')).toBeInTheDocument();
    expect(screen.getByText('Cells (4) · spread 13 mV')).toBeInTheDocument();
  });

  it('marks no cell as an outlier when they all read the same', async () => {
    withMetrics({
      ...METRICS,
      cells: [
        { index: 0, voltage: 3.33 },
        { index: 1, voltage: 3.33 },
      ],
      cellCount: 2,
    });

    render(<VoltgoBatteryBank refreshKey={0} />);

    await waitFor(() => expect(screen.getByLabelText('Cell 0 · 3.330 V')).toBeInTheDocument());
    expect(screen.queryByLabelText(/highest/)).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/lowest/)).not.toBeInTheDocument();
  });

  it('renders nothing when the controller is disabled', async () => {
    mockApi({});

    const { container } = render(<VoltgoBatteryBank refreshKey={0} />);

    await waitFor(() => expect(mockedGet).toHaveBeenCalledWith('/api/voltgo'));
    await waitFor(() => expect(container).toBeEmptyDOMElement());
    expect(mockedGet).not.toHaveBeenCalledWith('/api/voltgo/bank-a/metrics');
  });

  it('says it is waiting when no reading has arrived yet', async () => {
    mockApi({ ...index([BANK_A]), '/api/voltgo/bank-a/metrics': { status: 204 } });

    render(<VoltgoBatteryBank refreshKey={0} />);

    expect(
      await screen.findByText('Waiting for the first battery reading.'),
    ).toBeInTheDocument();
  });

  it('reports a failed fetch rather than rendering nothing', async () => {
    mockedGet.mockRejectedValue(httpError(500, 'Internal Server Error'));

    render(<VoltgoBatteryBank refreshKey={0} />);

    expect(
      await screen.findByText('Failed to load battery metrics: 500 Internal Server Error'),
    ).toBeInTheDocument();
  });

  it('refetches when refreshKey changes', async () => {
    withMetrics();

    const { rerender } = render(<VoltgoBatteryBank refreshKey={0} />);
    await waitFor(() => expect(screen.getByText('74 %')).toBeInTheDocument());

    withMetrics({ ...METRICS, soc: 81 });
    rerender(<VoltgoBatteryBank refreshKey={1} />);

    await waitFor(() => expect(screen.getByText('81 %')).toBeInTheDocument());
  });
  it('renders one panel per battery in the index', async () => {
    mockApi({
      ...index([BANK_A, BANK_B]),
      '/api/voltgo/bank-a/metrics': { status: 200, data: METRICS },
      '/api/voltgo/bank-a/info': { status: 200, data: INFO },
      '/api/voltgo/bank-b/metrics': { status: 200, data: { ...METRICS, soc: 41, voltage: 12.90 } },
      '/api/voltgo/bank-b/info': { status: 200, data: INFO },
    });

    render(<VoltgoBatteryBank refreshKey={0} />);

    // Each pack shows its own reading, so a divergent bank is visible rather
    // than being averaged away or overwritten by whichever answered last.
    await waitFor(() => expect(screen.getByText('74 %')).toBeInTheDocument());
    expect(screen.getByText('41 %')).toBeInTheDocument();
    expect(screen.getByText('13.31 V')).toBeInTheDocument();
    expect(screen.getByText('12.9 V')).toBeInTheDocument();

    expect(screen.getByText(/Battery Bank · bank-a/)).toBeInTheDocument();
    expect(screen.getByText(/Battery Bank · bank-b/)).toBeInTheDocument();
  });

  it('does not label the panel with an id when there is only one battery', async () => {
    withMetrics();

    render(<VoltgoBatteryBank refreshKey={0} />);

    await waitFor(() => expect(screen.getByText('74 %')).toBeInTheDocument());
    expect(screen.queryByText(/Battery Bank · /)).not.toBeInTheDocument();
  });

  it('renders the batteries that answer even when one is unreachable', async () => {
    mockApi({
      ...index([BANK_A, BANK_B]),
      '/api/voltgo/bank-a/metrics': { status: 200, data: METRICS },
      '/api/voltgo/bank-a/info': { status: 200, data: INFO },
      // bank-b is configured but has never connected.
      '/api/voltgo/bank-b/metrics': { status: 204 },
    });

    render(<VoltgoBatteryBank refreshKey={0} />);

    await waitFor(() => expect(screen.getByText('74 %')).toBeInTheDocument());
    expect(
      screen.getByText('Waiting for the first battery reading.'),
    ).toBeInTheDocument();
    expect(screen.getByText(/Battery Bank · bank-b/)).toBeInTheDocument();
  });

  it('renders nothing when the index lists no batteries', async () => {
    mockApi(index([]));

    const { container } = render(<VoltgoBatteryBank refreshKey={0} />);

    await waitFor(() => expect(mockedGet).toHaveBeenCalledWith('/api/voltgo'));
    expect(container).toBeEmptyDOMElement();
  });

  it('refetches the index as well as the panels when refreshKey changes', async () => {
    withMetrics();

    const { rerender } = render(<VoltgoBatteryBank refreshKey={0} />);
    await waitFor(() => expect(screen.getByText('74 %')).toBeInTheDocument());

    // A battery added to the configuration and a reload of the service shows
    // up as a longer index, and must produce a second panel.
    mockApi({
      ...index([BANK_A, BANK_B]),
      '/api/voltgo/bank-a/metrics': { status: 200, data: METRICS },
      '/api/voltgo/bank-a/info': { status: 200, data: INFO },
      '/api/voltgo/bank-b/metrics': { status: 200, data: { ...METRICS, soc: 41 } },
      '/api/voltgo/bank-b/info': { status: 200, data: INFO },
    });
    rerender(<VoltgoBatteryBank refreshKey={1} />);

    await waitFor(() => expect(screen.getByText('41 %')).toBeInTheDocument());
  });
});
