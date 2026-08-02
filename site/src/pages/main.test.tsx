import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi, type Mock } from 'vitest';

import Main from './main';
import type { EpeverMetrics, VoltgoMetrics } from '../api/types';

vi.mock('axios', () => {
  const isAxiosError = (error: unknown): boolean =>
    Boolean((error as { isAxiosError?: boolean } | null)?.isAxiosError);
  return {
    default: { get: vi.fn(), isAxiosError },
  };
});

import axios from 'axios';

const mockedGet = (axios as unknown as { get: Mock }).get;

const METRICS: EpeverMetrics = {
  timestamp: 1699000000,
  collectionTime: 0.42,
  arrayVoltage: 18.5,
  arrayCurrent: 2.1,
  arrayPower: 38.9,
  chargingCurrent: 2.9,
  chargingPower: 39.2,
  batteryVoltage: 13.4,
  batterySoc: 87,
  batteryTemp: 21.5,
  deviceTemp: 24.5,
  energyGeneratedDaily: 1.24,
  chargingStatus: 2,
};

const VOLTGO_METRICS: VoltgoMetrics = {
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

const notFound = (): Promise<never> => {
  const error = new Error('Request failed with status code 404') as Error & {
    isAxiosError: boolean;
    response: { status: number; statusText: string };
  };
  error.isAxiosError = true;
  error.response = { status: 404, statusText: 'Not Found' };
  return Promise.reject(error);
};

/**
 * Routes by URL rather than answering every call with the same payload: the
 * dashboard now fetches the voltgo battery endpoints too, and handing those the
 * epever payload would test a response the backend never sends. Voltgo is
 * absent unless a test says otherwise, which is what a solar-only deployment
 * looks like.
 */
function mockApi(responses: Record<string, unknown> = {}) {
  mockedGet.mockImplementation((url: string) => {
    if (url in responses) {
      return Promise.resolve({ status: 200, data: responses[url] });
    }
    if (url === '/api/epever/metrics') {
      return Promise.resolve({ status: 200, data: METRICS });
    }
    return notFound();
  });
}

const callCount = (url: string): number =>
  mockedGet.mock.calls.filter(call => call[0] === url).length;

beforeEach(() => {
  vi.clearAllMocks();
  mockApi();
});

describe('dashboard', () => {
  it('renders the fetched metric values', async () => {
    render(<Main />);

    await waitFor(() => expect(screen.getByText('38.9 W')).toBeInTheDocument());
    expect(screen.getByText('18.5 V')).toBeInTheDocument();
    expect(screen.getByText('2.1 A')).toBeInTheDocument();
    expect(screen.getByText('13.4 V')).toBeInTheDocument();
    expect(screen.getByText('87 %')).toBeInTheDocument();
    expect(screen.getByText('1.24 KWh')).toBeInTheDocument();
  });

  it.each([
    [0, 'Not charging'],
    [1, 'Float'],
    [2, 'Boost'],
    [3, 'Equalization'],
    [9, 'Unknown'],
  ])('renders charging status code %i as %s', async (code, label) => {
    mockApi({ '/api/epever/metrics': { ...METRICS, chargingStatus: code } });

    render(<Main />);

    await waitFor(() => expect(screen.getByText(label)).toBeInTheDocument());
  });

  it('shows an error rather than stale or blank values when the fetch fails', async () => {
    const error = new Error('Request failed') as Error & {
      isAxiosError: boolean;
      response: { status: number; statusText: string };
    };
    error.isAxiosError = true;
    error.response = { status: 500, statusText: 'Internal Server Error' };
    mockedGet.mockImplementation((url: string) =>
      url === '/api/epever/metrics' ? Promise.reject(error) : notFound(),
    );

    render(<Main />);

    expect(
      await screen.findByText('Failed to load metrics: 500 Internal Server Error'),
    ).toBeInTheDocument();
  });

  it('refetches when the refresh button is clicked', async () => {
    render(<Main />);
    await waitFor(() =>
      expect(mockedGet).toHaveBeenCalledWith('/api/epever/metrics'),
    );
    const before = callCount('/api/epever/metrics');

    screen.getByRole('button', { name: 'refresh metrics' }).click();

    await waitFor(() => expect(callCount('/api/epever/metrics')).toBe(before + 1));
  });

  it('hides the battery panel when the voltgo controller is not running', async () => {
    render(<Main />);

    await waitFor(() => expect(mockedGet).toHaveBeenCalledWith('/api/voltgo'));
    expect(screen.queryByText('Battery Bank')).not.toBeInTheDocument();
  });

  it('shows the battery panel and refreshes it alongside the dashboard', async () => {
    mockApi({
      '/api/voltgo': { batteries: [{ id: 'bank-a', address: 'A4:C1:37:43:A4:33' }] },
      '/api/voltgo/bank-a/metrics': VOLTGO_METRICS,
    });

    render(<Main />);
    await waitFor(() => expect(screen.getByText('Battery Bank')).toBeInTheDocument());

    const before = callCount('/api/voltgo/bank-a/metrics');

    screen.getByRole('button', { name: 'refresh metrics' }).click();

    await waitFor(() => expect(callCount('/api/voltgo/bank-a/metrics')).toBe(before + 1));
  });

  it('renders a panel per battery when several are configured', async () => {
    mockApi({
      '/api/voltgo': {
        batteries: [
          { id: 'bank-a', address: 'A4:C1:37:43:A4:33' },
          { id: 'bank-b', address: 'A4:C1:37:43:A4:42' },
        ],
      },
      '/api/voltgo/bank-a/metrics': VOLTGO_METRICS,
      '/api/voltgo/bank-b/metrics': { ...VOLTGO_METRICS, soc: 52 },
    });

    render(<Main />);

    await waitFor(() => expect(screen.getByText('Battery Bank · bank-a')).toBeInTheDocument());
    expect(screen.getByText('Battery Bank · bank-b')).toBeInTheDocument();
    expect(screen.getByText('52 %')).toBeInTheDocument();
  });
});
