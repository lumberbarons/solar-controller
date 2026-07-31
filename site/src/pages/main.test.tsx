import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi, type Mock } from 'vitest';

import Main from './main';
import type { EpeverMetrics } from '../api/types';

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

beforeEach(() => {
  vi.clearAllMocks();
});

describe('dashboard', () => {
  it('renders the fetched metric values', async () => {
    mockedGet.mockResolvedValue({ data: METRICS });

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
    mockedGet.mockResolvedValue({ data: { ...METRICS, chargingStatus: code } });

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
    mockedGet.mockRejectedValue(error);

    render(<Main />);

    expect(
      await screen.findByText('Failed to load metrics: 500 Internal Server Error'),
    ).toBeInTheDocument();
  });

  it('refetches when the refresh button is clicked', async () => {
    mockedGet.mockResolvedValue({ data: METRICS });

    render(<Main />);
    await waitFor(() => expect(mockedGet).toHaveBeenCalledTimes(1));

    screen.getByRole('button', { name: 'refresh metrics' }).click();

    await waitFor(() => expect(mockedGet).toHaveBeenCalledTimes(2));
  });
});
