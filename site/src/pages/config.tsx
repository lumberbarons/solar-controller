import React from 'react';

import axios from 'axios';
import { Box, Grid, Alert, Typography, Paper } from '@mui/material';
import { FormControl, InputLabel, Select, MenuItem, TextField, Button } from '@mui/material';

import {
  EDITABLE_DURATION_FIELDS,
  EDITABLE_VOLTAGE_FIELDS,
  type ApiError,
  type BatteryProfile,
  type BatteryProfileForm,
  type BatteryProfilePatch,
  type BatteryType,
  type ChargingParameters,
  type ChargingParametersForm,
  type ChargingParametersPatch,
} from '../api/types';

/**
 * Accepts both a text input's ChangeEvent and MUI's SelectChangeEvent, which
 * share the name/value shape this component reads but differ in type.
 */
type FieldChangeEvent = { target: { name: string; value: unknown } };

type ConfigState = {
  batteryProfile: BatteryProfileForm | undefined;
  originalBatteryProfile: BatteryProfile | undefined;
  chargingParameters: ChargingParametersForm | undefined;
  originalChargingParameters: ChargingParameters | undefined;
  loadError: string | undefined;
  saveError: string | undefined;
  successMessage: string | undefined;
  loading: boolean;
  batteryProfileSaved: boolean;
};

/** Reads `{"error": "..."}` out of a Gin error response, if present. */
function backendErrorMessage(error: unknown): string | undefined {
  if (axios.isAxiosError<ApiError>(error) && error.response) {
    return error.response.data?.error;
  }
  return undefined;
}

/**
 * Builds the "Failed to ...: reason" message for a request failure, preferring
 * the backend's own error text, then the HTTP status, then the transport error.
 * Every failure produces a message, so a rejected request can never leave the
 * screen silently unchanged.
 */
function requestErrorMessage(prefix: string, error: unknown): string {
  const backendError = backendErrorMessage(error);
  if (backendError) {
    return `${prefix}: ${backendError}`;
  }
  if (axios.isAxiosError(error) && error.response) {
    return `${prefix}: ${error.response.status} ${error.response.statusText}`;
  }
  const message = error instanceof Error ? error.message : String(error);
  return `${prefix}: ${message}`;
}

class Config extends React.Component<Record<string, never>, ConfigState> {
  private successTimer: ReturnType<typeof setTimeout> | null;

  constructor(props: Record<string, never>) {
    super(props);

    this.state = {
      batteryProfile: undefined,
      originalBatteryProfile: undefined,
      chargingParameters: undefined,
      originalChargingParameters: undefined,
      loadError: undefined,
      saveError: undefined,
      successMessage: undefined,
      loading: false,
      batteryProfileSaved: false
    };

    this.handleBatteryProfileSubmit = this.handleBatteryProfileSubmit.bind(this);
    this.handleChargingParametersSubmit = this.handleChargingParametersSubmit.bind(this);
    this.handleBatteryProfileChange = this.handleBatteryProfileChange.bind(this);
    this.handleChargingParametersChange = this.handleChargingParametersChange.bind(this);
    this.fetchConfig = this.fetchConfig.bind(this);

    this.successTimer = null;
  }

  componentDidMount() {
    this.fetchConfig();
  }

  componentWillUnmount() {
    // Clear timer when component unmounts
    if (this.successTimer) {
      clearTimeout(this.successTimer);
    }
  }

  setSuccessMessage(message: string) {
    // Clear any existing timer
    if (this.successTimer) {
      clearTimeout(this.successTimer);
    }

    // Set the success message
    this.setState({ successMessage: message, saveError: undefined });

    // Auto-dismiss after 4 seconds
    this.successTimer = setTimeout(() => {
      this.setState({ successMessage: undefined });
      this.successTimer = null;
    }, 4000);
  }

  /** Drops a pending auto-dismiss so an error is not cleared by an earlier success. */
  clearSuccessTimer() {
    if (this.successTimer) {
      clearTimeout(this.successTimer);
      this.successTimer = null;
    }
  }

  fetchConfig() {
    this.setState({ loading: true });

    // Fetch both battery profile and charging parameters
    Promise.all([
      axios.get<BatteryProfile>(`/api/epever/battery-profile`),
      axios.get<ChargingParameters>(`/api/epever/charging-parameters`)
    ])
      .then(([profileRes, paramsRes]) => {
        this.setState({
          originalBatteryProfile: structuredClone(profileRes.data),
          batteryProfile: profileRes.data,
          originalChargingParameters: structuredClone(paramsRes.data),
          chargingParameters: paramsRes.data,
          loadError: undefined,
          saveError: undefined,
          successMessage: undefined,
          loading: false,
          batteryProfileSaved: profileRes.data.batteryType === 'userDefined'
        });
      }).catch(error => {
        console.error('Failed to load configuration:', error);
        this.setState({
          loadError: requestErrorMessage('Failed to load configuration', error),
          loading: false
        });
      });
  }

  handleBatteryProfileChange(event: FieldChangeEvent) {
    const name = event.target.name as keyof BatteryProfileForm;
    const value = event.target.value as BatteryProfileForm[typeof name];

    this.setState(prev => (
      prev.batteryProfile
        ? { batteryProfile: { ...prev.batteryProfile, [name]: value } }
        : null
    ));
  }

  handleChargingParametersChange(event: FieldChangeEvent) {
    const name = event.target.name as keyof ChargingParametersForm;
    const value = event.target.value as ChargingParametersForm[typeof name];

    this.setState(prev => (
      prev.chargingParameters
        ? { chargingParameters: { ...prev.chargingParameters, [name]: value } }
        : null
    ));
  }

  handleBatteryProfileSubmit(event: React.FormEvent) {
    event.preventDefault();

    const original = this.state.originalBatteryProfile;
    const current = this.state.batteryProfile;
    if (!original || !current) {
      return;
    }

    // Send only what changed, so an untouched field is never rewritten to EEPROM
    const payload: BatteryProfilePatch = {};

    if (current.batteryType !== original.batteryType) {
      payload.batteryType = current.batteryType as BatteryType;
    }

    if (current.batteryCapacity !== original.batteryCapacity) {
      payload.batteryCapacity = parseInt(String(current.batteryCapacity));
    }

    if (current.tempCompCoefficient !== original.tempCompCoefficient) {
      payload.tempCompCoefficient = parseFloat(String(current.tempCompCoefficient));
    }

    axios.patch<BatteryProfile>(`/api/epever/battery-profile`, payload)
      .then(res => {
        this.setState({
          originalBatteryProfile: structuredClone(res.data),
          batteryProfile: res.data,
          saveError: undefined,
          batteryProfileSaved: res.data.batteryType === 'userDefined'
        });
        this.setSuccessMessage('Battery profile saved successfully!');

        // Auto-refresh charging parameters after battery profile save
        return axios.get<ChargingParameters>(`/api/epever/charging-parameters`);
      })
      .then(paramsRes => {
        this.setState({
          originalChargingParameters: structuredClone(paramsRes.data),
          chargingParameters: paramsRes.data
        });
      })
      .catch(error => {
        this.setState({
          saveError: requestErrorMessage('Failed to save battery profile', error),
          successMessage: undefined
        });
        this.clearSuccessTimer();
      });
  }

  handleChargingParametersSubmit(event: React.FormEvent) {
    event.preventDefault();

    const original = this.state.originalChargingParameters;
    const current = this.state.chargingParameters;
    if (!original || !current) {
      return;
    }

    // Send only changed fields. Durations are whole minutes or days on the
    // device, so they are parsed as integers; everything else is a voltage.
    const payload: ChargingParametersPatch = {};

    for (const field of EDITABLE_DURATION_FIELDS) {
      if (current[field] !== original[field]) {
        payload[field] = parseInt(String(current[field]));
      }
    }

    for (const field of EDITABLE_VOLTAGE_FIELDS) {
      if (current[field] !== original[field]) {
        payload[field] = parseFloat(String(current[field]));
      }
    }

    axios.patch<ChargingParameters>(`/api/epever/charging-parameters`, payload)
      .then(res => {
        this.setState({
          originalChargingParameters: structuredClone(res.data),
          chargingParameters: res.data,
          saveError: undefined
        });
        this.setSuccessMessage('Charging parameters saved successfully!');
      }).catch(error => {
        this.setState({
          saveError: requestErrorMessage('Failed to save charging parameters', error),
          successMessage: undefined
        });
        this.clearSuccessTimer();
      });
  }

  render() {
    const batteryProfile: BatteryProfileForm = this.state.batteryProfile || { batteryType: '', batteryCapacity: '', tempCompCoefficient: '' };
    const chargingParameters: ChargingParametersForm = this.state.chargingParameters || {
      boostDuration: '', equalizationCycle: '', equalizationDuration: '',
      boostVoltage: '', boostReconnectChargingVoltage: '', floatVoltage: '',
      equalizationVoltage: '', chargingLimitVoltage: '', overVoltDisconnectVoltage: '',
      overVoltReconnectVoltage: '', lowVoltDisconnectVoltage: '', lowVoltReconnectVoltage: '',
      underVoltWarningVoltage: '', underVoltWarningReconnectVoltage: '', dischargingLimitVoltage: '',
      batteryTempUpperLimit: '', batteryTempLowerLimit: '',
      controllerTempUpperLimit: '', controllerTempLowerLimit: ''
    };
    const isUserDefined = batteryProfile.batteryType === 'userDefined';
    const hasLoadError = !!this.state.loadError;
    const hasSaveError = !!this.state.saveError;
    const hasSuccessMessage = !!this.state.successMessage;
    const canEditChargingParams = isUserDefined && this.state.batteryProfileSaved;

    return (
      <Box sx={{ width: '100%', p: 1.5, mt: 1, backgroundColor: 'white', minHeight: 'calc(100vh - 72px)', maxWidth: 1400, mx: 'auto' }}>
        {/* Error Alerts */}
        {hasLoadError && (
          <Alert severity="error" sx={{ mb: 1, py: 0.5 }}>
            {this.state.loadError}
          </Alert>
        )}
        {hasSaveError && (
          <Alert severity="error" sx={{ mb: 1, py: 0.5 }}>
            {this.state.saveError}
          </Alert>
        )}
        {/* Success Alert */}
        {hasSuccessMessage && (
          <Alert severity="success" sx={{ mb: 1, py: 0.5 }}>
            {this.state.successMessage}
          </Alert>
        )}
          {/* Battery Profile Section */}
          <Box
            component="form"
            autoComplete="off"
            onSubmit={this.handleBatteryProfileSubmit}
            sx={{ mb: 2 }}
          >
            <Typography variant="subtitle1" sx={{ mb: 0.75, fontWeight: 600, color: '#1b5e20' }}>
              Battery Profile
            </Typography>
            <Paper elevation={2} sx={{ p: 1.5, mb: 1, borderRadius: 2 }}>
              <Grid container spacing={1.5}>
                <Grid size={{ xs: 12, sm: 6 }}>
                  <FormControl fullWidth>
                    <InputLabel>Battery Type</InputLabel>
                    <Select
                      name="batteryType"
                      value={batteryProfile.batteryType}
                      label="Battery Type"
                      onChange={this.handleBatteryProfileChange}
                    >
                      <MenuItem value="sealed">Sealed</MenuItem>
                      <MenuItem value="gel">Gel</MenuItem>
                      <MenuItem value="flooded">Flooded</MenuItem>
                      <MenuItem value="userDefined">User Defined</MenuItem>
                    </Select>
                  </FormControl>
                </Grid>
                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    required
                    fullWidth
                    label="Battery Capacity (Ah)"
                    name="batteryCapacity"
                    value={batteryProfile.batteryCapacity}
                    onChange={this.handleBatteryProfileChange}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    required
                    fullWidth
                    label="Temp Comp Coeff (mV/°C/2V)"
                    name="tempCompCoefficient"
                    value={batteryProfile.tempCompCoefficient}
                    onChange={this.handleBatteryProfileChange}
                  />
                </Grid>
              </Grid>
            </Paper>

            <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
              <Button
                type="submit"
                variant="contained"
                color="primary"
                size="small"
                disabled={hasLoadError}
                sx={{
                  backgroundColor: '#2e7d32',
                  '&:hover': {
                    backgroundColor: '#1b5e20'
                  },
                  '&:disabled': {
                    backgroundColor: '#9e9e9e'
                  },
                  px: 2,
                  py: 0.75,
                  fontWeight: 600
                }}
              >
                Save Battery Profile
              </Button>
            </Box>
          </Box>

          {/* Charging Parameters Section */}
          <Box
            component="form"
            autoComplete="off"
            onSubmit={this.handleChargingParametersSubmit}
          >
            <Typography variant="subtitle1" sx={{ mb: 0.75, fontWeight: 600, color: '#1b5e20' }}>
              Charging Parameters
            </Typography>

            {!canEditChargingParams && (
              <Alert severity="info" sx={{ mb: 1, py: 0.5 }}>
                {!isUserDefined
                  ? "Set Battery Type to 'User Defined' and save to edit charging parameters"
                  : "Save battery profile with 'User Defined' type to edit charging parameters"}
              </Alert>
            )}

            {/* Charging Voltage Settings */}
            <Typography variant="body2" sx={{ mb: 0.5, fontWeight: 500, color: '#424242' }}>
              Charging Voltage Settings
            </Typography>
            <Paper elevation={2} sx={{ p: 1.5, mb: 1, borderRadius: 2 }}>
              <Grid container spacing={1.5}>
                <Grid size={{ xs: 12, sm: 6, md: 4 }}>
                  <TextField
                    required
                    fullWidth
                    disabled={!canEditChargingParams}
                    label="Charging Limit Voltage"
                    name="chargingLimitVoltage"
                    value={chargingParameters.chargingLimitVoltage}
                    onChange={this.handleChargingParametersChange}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6, md: 4 }}>
                  <TextField
                    required
                    fullWidth
                    disabled={!canEditChargingParams}
                    label="Boost Voltage"
                    name="boostVoltage"
                    value={chargingParameters.boostVoltage}
                    onChange={this.handleChargingParametersChange}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6, md: 4 }}>
                  <TextField
                    required
                    fullWidth
                    disabled={!canEditChargingParams}
                    label="Boost Reconnect Voltage"
                    name="boostReconnectChargingVoltage"
                    value={chargingParameters.boostReconnectChargingVoltage}
                    onChange={this.handleChargingParametersChange}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6, md: 4 }}>
                  <TextField
                    required
                    fullWidth
                    disabled={!canEditChargingParams}
                    label="Boost Duration (min)"
                    name="boostDuration"
                    value={chargingParameters.boostDuration}
                    onChange={this.handleChargingParametersChange}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6, md: 4 }}>
                  <TextField
                    required
                    fullWidth
                    disabled={!canEditChargingParams}
                    label="Float Voltage"
                    name="floatVoltage"
                    value={chargingParameters.floatVoltage}
                    onChange={this.handleChargingParametersChange}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6, md: 4 }}>
                  <TextField
                    required
                    fullWidth
                    disabled={!canEditChargingParams}
                    label="Equalization Voltage"
                    name="equalizationVoltage"
                    value={chargingParameters.equalizationVoltage}
                    onChange={this.handleChargingParametersChange}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6, md: 4 }}>
                  <TextField
                    required
                    fullWidth
                    disabled={!canEditChargingParams}
                    label="Equalization Cycle (days)"
                    name="equalizationCycle"
                    value={chargingParameters.equalizationCycle}
                    onChange={this.handleChargingParametersChange}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6, md: 4 }}>
                  <TextField
                    required
                    fullWidth
                    disabled={!canEditChargingParams}
                    label="Equalization Duration (min)"
                    name="equalizationDuration"
                    value={chargingParameters.equalizationDuration}
                    onChange={this.handleChargingParametersChange}
                  />
                </Grid>
              </Grid>
            </Paper>

            {/* Protection Voltage Settings */}
            <Typography variant="body2" sx={{ mb: 0.5, fontWeight: 500, color: '#424242' }}>
              Protection Voltage Settings
            </Typography>
            <Paper elevation={2} sx={{ p: 1.5, mb: 1, borderRadius: 2 }}>
              <Grid container spacing={1.5}>
                <Grid size={{ xs: 12, sm: 6, md: 3 }}>
                  <TextField
                    required
                    fullWidth
                    disabled={!canEditChargingParams}
                    label="Over Volt Disconnect"
                    name="overVoltDisconnectVoltage"
                    value={chargingParameters.overVoltDisconnectVoltage}
                    onChange={this.handleChargingParametersChange}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6, md: 3 }}>
                  <TextField
                    required
                    fullWidth
                    disabled={!canEditChargingParams}
                    label="Over Volt Reconnect"
                    name="overVoltReconnectVoltage"
                    value={chargingParameters.overVoltReconnectVoltage}
                    onChange={this.handleChargingParametersChange}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6, md: 3 }}>
                  <TextField
                    required
                    fullWidth
                    disabled={!canEditChargingParams}
                    label="Low Volt Disconnect"
                    name="lowVoltDisconnectVoltage"
                    value={chargingParameters.lowVoltDisconnectVoltage}
                    onChange={this.handleChargingParametersChange}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6, md: 3 }}>
                  <TextField
                    required
                    fullWidth
                    disabled={!canEditChargingParams}
                    label="Low Volt Reconnect"
                    name="lowVoltReconnectVoltage"
                    value={chargingParameters.lowVoltReconnectVoltage}
                    onChange={this.handleChargingParametersChange}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6, md: 4 }}>
                  <TextField
                    required
                    fullWidth
                    disabled={!canEditChargingParams}
                    label="Under Volt Warning"
                    name="underVoltWarningVoltage"
                    value={chargingParameters.underVoltWarningVoltage}
                    onChange={this.handleChargingParametersChange}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6, md: 4 }}>
                  <TextField
                    required
                    fullWidth
                    disabled={!canEditChargingParams}
                    label="Under Volt Reconnect"
                    name="underVoltWarningReconnectVoltage"
                    value={chargingParameters.underVoltWarningReconnectVoltage}
                    onChange={this.handleChargingParametersChange}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6, md: 4 }}>
                  <TextField
                    required
                    fullWidth
                    disabled={!canEditChargingParams}
                    label="Discharging Limit Voltage"
                    name="dischargingLimitVoltage"
                    value={chargingParameters.dischargingLimitVoltage}
                    onChange={this.handleChargingParametersChange}
                  />
                </Grid>
              </Grid>
            </Paper>

            <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
              <Button
                type="submit"
                variant="contained"
                color="primary"
                size="small"
                disabled={!canEditChargingParams || hasLoadError}
                sx={{
                  backgroundColor: '#2e7d32',
                  '&:hover': {
                    backgroundColor: '#1b5e20'
                  },
                  '&:disabled': {
                    backgroundColor: '#9e9e9e'
                  },
                  px: 2,
                  py: 0.75,
                  fontWeight: 600
                }}
              >
                Save Charging Parameters
              </Button>
            </Box>
          </Box>
        </Box>
      );
  }
}

export default Config
