import { useCallback, useEffect, useState } from 'react';

import axios from 'axios';
import { Alert, Box, Chip, Grid, Stack, Typography } from '@mui/material';

import Metric from './metric';
import type { VoltgoBatteryRef, VoltgoCell, VoltgoIndex, VoltgoInfo, VoltgoMetrics } from '../api/types';

/**
 * Below this many amps the pack is reported as idle rather than charging or
 * discharging: a resting pack still reads a few tens of milliamps, and flipping
 * the label on that noise reads as a fault to anyone watching the dashboard.
 */
const IDLE_CURRENT_THRESHOLD_A = 0.05;

type PanelState =
  /** The first fetch has not settled yet. */
  | { kind: 'loading' }
  /** This battery's routes are gone: it was removed from the configuration. */
  | { kind: 'absent' }
  /** Configured, but no successful collection has happened yet (204). */
  | { kind: 'pending' }
  | { kind: 'ready'; metrics: VoltgoMetrics }
  | { kind: 'error'; message: string };

type BankState =
  | { kind: 'loading' }
  /** No voltgo controller is running: the index endpoint is not registered. */
  | { kind: 'absent' }
  | { kind: 'ready'; batteries: VoltgoBatteryRef[] }
  | { kind: 'error'; message: string };

type RefreshProps = {
  /** Changing this triggers a refetch, so the dashboard refresh covers this panel too. */
  refreshKey: number;
};

function describeFlow(current: number): string {
  if (current > IDLE_CURRENT_THRESHOLD_A) {
    return 'Charging';
  }
  if (current < -IDLE_CURRENT_THRESHOLD_A) {
    return 'Discharging';
  }
  return 'Idle';
}

/** Signed, so the direction is readable from the number as well as the label. */
function formatCurrent(current: number): string {
  return `${current > 0 ? '+' : ''}${current.toFixed(2)}`;
}

function cellExtremes(cells: VoltgoCell[]): { min?: number; max?: number; spreadMv: number } {
  if (cells.length === 0) {
    return { spreadMv: 0 };
  }
  const voltages = cells.map(cell => cell.voltage);
  const min = Math.min(...voltages);
  const max = Math.max(...voltages);
  return { min, max, spreadMv: (max - min) * 1000 };
}

function errorMessage(error: unknown): string {
  if (axios.isAxiosError(error) && error.response) {
    return `Failed to load battery metrics: ${error.response.status} ${error.response.statusText}`;
  }
  return `Failed to load battery metrics: ${(error as Error).message}`;
}

type VoltgoBatteryPanelProps = RefreshProps & {
  battery: VoltgoBatteryRef;
  /**
   * With one battery the id is noise — often just its BLE address. With
   * several it is the only thing telling the panels apart, so it is shown.
   */
  showId: boolean;
};

/** One battery pack, read from its own per-battery endpoints. */
function VoltgoBatteryPanel({ battery, showId, refreshKey }: VoltgoBatteryPanelProps) {
  const [state, setState] = useState<PanelState>({ kind: 'loading' });
  const [info, setInfo] = useState<VoltgoInfo | null>(null);

  const { id } = battery;

  const fetchInfo = useCallback(() => {
    axios.get<VoltgoInfo>(`/api/voltgo/${id}/info`)
      .then(response => {
        // 204 until the first connection has read the static info.
        setInfo(response.status === 204 ? null : response.data);
      })
      .catch(() => {
        // Info is decoration; the panel is useful without it.
        setInfo(null);
      });
  }, [id]);

  useEffect(() => {
    let cancelled = false;

    axios.get<VoltgoMetrics>(`/api/voltgo/${id}/metrics`)
      .then(response => {
        if (cancelled) {
          return;
        }
        if (response.status === 204) {
          setState({ kind: 'pending' });
          return;
        }
        setState({ kind: 'ready', metrics: response.data });
        fetchInfo();
      })
      .catch(error => {
        if (cancelled) {
          return;
        }
        if (axios.isAxiosError(error) && error.response?.status === 404) {
          setState({ kind: 'absent' });
          return;
        }
        console.error(`Failed to load voltgo battery ${id} metrics:`, error);
        setState({ kind: 'error', message: errorMessage(error) });
      });

    return () => {
      cancelled = true;
    };
  }, [id, refreshKey, fetchInfo]);

  if (state.kind === 'absent' || state.kind === 'loading') {
    return null;
  }

  const heading = (
    <Typography variant="h6" sx={{ mb: 1, fontWeight: 600, color: '#1b5e20' }}>
      {showId ? `Battery Bank · ${id}` : 'Battery Bank'}
      {info && (
        <Typography component="span" sx={{ ml: 1, fontSize: '0.875rem', color: 'text.secondary' }}>
          {info.chemistry} · {info.nominalVoltage} V · {info.capacityAh} Ah
        </Typography>
      )}
    </Typography>
  );

  if (state.kind === 'error') {
    return (
      <>
        {heading}
        <Alert severity="error" sx={{ mb: 2 }}>{state.message}</Alert>
      </>
    );
  }

  if (state.kind === 'pending') {
    return (
      <>
        {heading}
        <Alert severity="info" sx={{ mb: 2 }}>Waiting for the first battery reading.</Alert>
      </>
    );
  }

  const { metrics } = state;
  const cells = metrics.cells ?? [];
  const { min, max, spreadMv } = cellExtremes(cells);
  const hasSpread = min !== undefined && max !== undefined && max > min;

  return (
    <>
      {heading}
      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Metric title="Battery SOC" value={metrics.soc} unit="%" />
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Metric title="Battery Voltage" value={metrics.voltage} unit="V" />
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Metric title="Battery Current" value={formatCurrent(metrics.current)} unit="A" />
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Metric title="Battery Flow" value={describeFlow(metrics.current)} unit="" />
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Metric title="Battery Health" value={metrics.soh} unit="%" />
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Metric title="Battery Temp" value={metrics.temperature} unit="°C" />
        </Grid>
      </Grid>

      {cells.length > 0 && (
        <Box sx={{ mb: 2 }}>
          <Typography sx={{ mb: 1, fontWeight: 600, color: '#1b5e20' }}>
            Cells ({metrics.cellCount || cells.length}) · spread {spreadMv.toFixed(0)} mV
          </Typography>
          <Stack direction="row" spacing={1} useFlexGap sx={{ flexWrap: 'wrap' }}>
            {cells.map(cell => {
              // Indices are the device's own, matching the `cell` label on the
              // voltgo_cell_voltage Prometheus metric.
              const label = `Cell ${cell.index} · ${cell.voltage.toFixed(3)} V`;
              // A pack whose cells all read the same has no outlier to point
              // at, so highlighting every chip would be noise.
              const isMax = hasSpread && cell.voltage === max;
              const isMin = hasSpread && cell.voltage === min;
              const extreme = isMax ? ' (highest)' : isMin ? ' (lowest)' : '';
              const borderColor = isMax ? '#c62828' : isMin ? '#1565c0' : 'rgba(0,0,0,0.23)';

              return (
                <Chip
                  key={cell.index}
                  label={label}
                  aria-label={`${showId ? `${id} ` : ''}${label}${extreme}`}
                  variant="outlined"
                  sx={{
                    borderColor,
                    borderWidth: isMax || isMin ? 2 : 1,
                    fontWeight: isMax || isMin ? 600 : 400,
                  }}
                />
              );
            })}
          </Stack>
        </Box>
      )}
    </>
  );
}

/**
 * Every battery pack reported by the voltgo BLE controller.
 *
 * The batteries come from the index endpoint rather than being hardcoded, so
 * a deployment with four packs renders four panels without a frontend change.
 * Renders nothing when the controller is disabled — the index is then
 * unregistered and answers 404 — so a solar-only deployment sees no empty panel.
 */
function VoltgoBatteryBank({ refreshKey }: RefreshProps) {
  const [state, setState] = useState<BankState>({ kind: 'loading' });

  useEffect(() => {
    let cancelled = false;

    axios.get<VoltgoIndex>('/api/voltgo')
      .then(response => {
        if (cancelled) {
          return;
        }
        setState({ kind: 'ready', batteries: response.data?.batteries ?? [] });
      })
      .catch(error => {
        if (cancelled) {
          return;
        }
        if (axios.isAxiosError(error) && error.response?.status === 404) {
          setState({ kind: 'absent' });
          return;
        }
        console.error('Failed to load the voltgo battery list:', error);
        setState({ kind: 'error', message: errorMessage(error) });
      });

    return () => {
      cancelled = true;
    };
  }, [refreshKey]);

  if (state.kind === 'absent' || state.kind === 'loading') {
    return null;
  }

  if (state.kind === 'error') {
    return <Alert severity="error" sx={{ mb: 2 }}>{state.message}</Alert>;
  }

  const { batteries } = state;
  const showId = batteries.length > 1;

  return (
    <>
      {batteries.map(battery => (
        <VoltgoBatteryPanel
          key={battery.id}
          battery={battery}
          showId={showId}
          refreshKey={refreshKey}
        />
      ))}
    </>
  );
}

export default VoltgoBatteryBank;
