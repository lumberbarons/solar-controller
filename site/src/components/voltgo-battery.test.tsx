import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi, type Mock } from 'vitest';

import VoltgoBattery from './voltgo-battery';
import type { VoltgoInfo, VoltgoMetrics } from '../api/types';

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

const withMetrics = (metrics: VoltgoMetrics = METRICS) =>
  mockApi({
    '/api/voltgo/metrics': { status: 200, data: metrics },
    '/api/voltgo/info': { status: 200, data: INFO },
  });

beforeEach(() => {
  vi.clearAllMocks();
});

describe('voltgo battery panel', () => {
  it('renders the pack metrics', async () => {
    withMetrics();

    render(<VoltgoBattery refreshKey={0} />);

    await waitFor(() => expect(screen.getByText('74 %')).toBeInTheDocument());
    expect(screen.getByText('13.31 V')).toBeInTheDocument();
    expect(screen.getByText('99 %')).toBeInTheDocument();
    expect(screen.getByText('19.5 °C')).toBeInTheDocument();
  });

  it('shows the battery description once the info endpoint answers', async () => {
    withMetrics();

    render(<VoltgoBattery refreshKey={0} />);

    await waitFor(() =>
      expect(screen.getByText('LiFePO4 · 12.8 V · 100 Ah')).toBeInTheDocument(),
    );
  });

  it('still renders the pack when the info endpoint has no data yet', async () => {
    mockApi({
      '/api/voltgo/metrics': { status: 200, data: METRICS },
      '/api/voltgo/info': { status: 204 },
    });

    render(<VoltgoBattery refreshKey={0} />);

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

    render(<VoltgoBattery refreshKey={0} />);

    await waitFor(() => expect(screen.getByText(displayed)).toBeInTheDocument());
    expect(screen.getByText(flow)).toBeInTheDocument();
  });

  it('highlights the highest and lowest cell and reports the spread', async () => {
    withMetrics();

    render(<VoltgoBattery refreshKey={0} />);

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

    render(<VoltgoBattery refreshKey={0} />);

    await waitFor(() => expect(screen.getByLabelText('Cell 0 · 3.330 V')).toBeInTheDocument());
    expect(screen.queryByLabelText(/highest/)).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/lowest/)).not.toBeInTheDocument();
  });

  it('renders nothing when the controller is disabled', async () => {
    mockApi({});

    const { container } = render(<VoltgoBattery refreshKey={0} />);

    await waitFor(() => expect(mockedGet).toHaveBeenCalledWith('/api/voltgo/metrics'));
    await waitFor(() => expect(container).toBeEmptyDOMElement());
    expect(mockedGet).not.toHaveBeenCalledWith('/api/voltgo/info');
  });

  it('says it is waiting when no reading has arrived yet', async () => {
    mockApi({ '/api/voltgo/metrics': { status: 204 } });

    render(<VoltgoBattery refreshKey={0} />);

    expect(
      await screen.findByText('Waiting for the first battery reading.'),
    ).toBeInTheDocument();
  });

  it('reports a failed fetch rather than rendering nothing', async () => {
    mockedGet.mockRejectedValue(httpError(500, 'Internal Server Error'));

    render(<VoltgoBattery refreshKey={0} />);

    expect(
      await screen.findByText('Failed to load battery metrics: 500 Internal Server Error'),
    ).toBeInTheDocument();
  });

  it('refetches when refreshKey changes', async () => {
    withMetrics();

    const { rerender } = render(<VoltgoBattery refreshKey={0} />);
    await waitFor(() => expect(screen.getByText('74 %')).toBeInTheDocument());

    withMetrics({ ...METRICS, soc: 81 });
    rerender(<VoltgoBattery refreshKey={1} />);

    await waitFor(() => expect(screen.getByText('81 %')).toBeInTheDocument());
  });
});
