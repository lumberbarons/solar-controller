import React from 'react';

import axios from 'axios';
import { Box, Grid, Alert, Typography, Paper } from '@mui/material';
import { FormControl, InputLabel, Select, MenuItem, TextField, Button } from '@mui/material';

class Config extends React.Component {
  constructor(props) {
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

  setSuccessMessage(message) {
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

  fetchConfig() {
    this.setState({loading: true});

    // Fetch both battery profile and charging parameters
    Promise.all([
      axios.get(`/api/solar/battery-profile`),
      axios.get(`/api/solar/charging-parameters`)
    ])
      .then(([profileRes, paramsRes]) => {
        let profileClone = JSON.parse(JSON.stringify(profileRes.data));
        let paramsClone = JSON.parse(JSON.stringify(paramsRes.data));
        this.setState({
          originalBatteryProfile: profileClone,
          batteryProfile: profileRes.data,
          originalChargingParameters: paramsClone,
          chargingParameters: paramsRes.data,
          loadError: undefined,
          saveError: undefined,
          successMessage: undefined,
          loading: false,
          batteryProfileSaved: profileRes.data.batteryType === 'userDefined'
        });
      }).catch(error => {
        console.error(JSON.stringify(error));
        const errorMessage = error.response
          ? `Failed to load configuration: ${error.response.status} ${error.response.statusText}`
          : `Failed to load configuration: ${error.message}`;
        this.setState({
          loadError: errorMessage,
          loading: false
        });
      });
  }

  handleBatteryProfileChange(event) {
    const value = event.target.value;
    const name = event.target.name;

    let batteryProfile = this.state.batteryProfile;
    batteryProfile[name] = value;

    this.setState({
      batteryProfile: batteryProfile
    });
  }

  handleChargingParametersChange(event) {
    const value = event.target.value;
    const name = event.target.name;

    let chargingParameters = this.state.chargingParameters;
    chargingParameters[name] = value;

    this.setState({
      chargingParameters: chargingParameters
    });
  }

  handleBatteryProfileSubmit(event) {
    const payload = {};

    const original = this.state.originalBatteryProfile;
    const current = this.state.batteryProfile;

    if(current.batteryType !== original.batteryType) {
      payload.batteryType = current.batteryType;
    }

    if(current.batteryCapacity !== original.batteryCapacity) {
      payload.batteryCapacity = parseInt(current.batteryCapacity);
    }

    axios.patch(`/api/solar/battery-profile`, payload)
      .then(res => {
        let profileClone = JSON.parse(JSON.stringify(res.data));
        const isUserDefined = res.data.batteryType === 'userDefined';
        this.setState({
          originalBatteryProfile: profileClone,
          batteryProfile: res.data,
          saveError: undefined,
          batteryProfileSaved: isUserDefined
        });
        this.setSuccessMessage('Battery profile saved successfully!');

        // Auto-refresh charging parameters after battery profile save
        return axios.get(`/api/solar/charging-parameters`);
      })
      .then(paramsRes => {
        let paramsClone = JSON.parse(JSON.stringify(paramsRes.data));
        this.setState({
          originalChargingParameters: paramsClone,
          chargingParameters: paramsRes.data
        });
      })
      .catch(error => {
        let errorMessage;
        if (error.response) {
          // Check if backend returned a detailed error message
          const backendError = error.response.data?.error;
          if (backendError) {
            errorMessage = `Failed to save battery profile: ${backendError}`;
          } else {
            errorMessage = `Failed to save battery profile: ${error.response.status} ${error.response.statusText}`;
          }
        } else {
          errorMessage = `Failed to save battery profile: ${error.message}`;
        }
        this.setState({saveError: errorMessage, successMessage: undefined});
        // Clear success timer on error
        if (this.successTimer) {
          clearTimeout(this.successTimer);
          this.successTimer = null;
        }
      });

    event.preventDefault();
  }

  handleChargingParametersSubmit(event) {
    const payload = {};

    const original = this.state.originalChargingParameters;
    const current = this.state.chargingParameters;

    // Track changes for all charging parameter fields
    if(current.equalizationCycle !== original.equalizationCycle) {
      payload.equalizationCycle = parseInt(current.equalizationCycle);
    }

    if(current.equalizationVoltage !== original.equalizationVoltage) {
      payload.equalizationVoltage = parseFloat(current.equalizationVoltage);
    }

    if(current.equalizationDuration !== original.equalizationDuration) {
      payload.equalizationDuration = parseInt(current.equalizationDuration);
    }

    if(current.boostVoltage !== original.boostVoltage) {
      payload.boostVoltage = parseFloat(current.boostVoltage);
    }

    if(current.boostDuration !== original.boostDuration) {
      payload.boostDuration = parseInt(current.boostDuration);
    }

    if(current.floatVoltage !== original.floatVoltage) {
      payload.floatVoltage = parseFloat(current.floatVoltage);
    }

    if(current.chargingLimitVoltage !== original.chargingLimitVoltage) {
      payload.chargingLimitVoltage = parseFloat(current.chargingLimitVoltage);
    }

    if(current.boostReconnectChargingVoltage !== original.boostReconnectChargingVoltage) {
      payload.boostReconnectChargingVoltage = parseFloat(current.boostReconnectChargingVoltage);
    }

    if(current.overVoltDisconnectVoltage !== original.overVoltDisconnectVoltage) {
      payload.overVoltDisconnectVoltage = parseFloat(current.overVoltDisconnectVoltage);
    }

    if(current.overVoltReconnectVoltage !== original.overVoltReconnectVoltage) {
      payload.overVoltReconnectVoltage = parseFloat(current.overVoltReconnectVoltage);
    }

    if(current.lowVoltDisconnectVoltage !== original.lowVoltDisconnectVoltage) {
      payload.lowVoltDisconnectVoltage = parseFloat(current.lowVoltDisconnectVoltage);
    }

    if(current.lowVoltReconnectVoltage !== original.lowVoltReconnectVoltage) {
      payload.lowVoltReconnectVoltage = parseFloat(current.lowVoltReconnectVoltage);
    }

    if(current.underVoltWarningVoltage !== original.underVoltWarningVoltage) {
      payload.underVoltWarningVoltage = parseFloat(current.underVoltWarningVoltage);
    }

    if(current.underVoltWarningReconnectVoltage !== original.underVoltWarningReconnectVoltage) {
      payload.underVoltWarningReconnectVoltage = parseFloat(current.underVoltWarningReconnectVoltage);
    }

    if(current.dischargingLimitVoltage !== original.dischargingLimitVoltage) {
      payload.dischargingLimitVoltage = parseFloat(current.dischargingLimitVoltage);
    }

    axios.patch(`/api/solar/charging-parameters`, payload)
      .then(res => {
        let paramsClone = JSON.parse(JSON.stringify(res.data));
        this.setState({
          originalChargingParameters: paramsClone,
          chargingParameters: res.data,
          saveError: undefined
        });
        this.setSuccessMessage('Charging parameters saved successfully!');
      }).catch(error => {
        let errorMessage;
        if (error.response) {
          // Check if backend returned a detailed error message
          const backendError = error.response.data?.error;
          if (backendError) {
            errorMessage = `Failed to save charging parameters: ${backendError}`;
          } else {
            errorMessage = `Failed to save charging parameters: ${error.response.status} ${error.response.statusText}`;
          }
        } else {
          errorMessage = `Failed to save charging parameters: ${error.message}`;
        }
        this.setState({saveError: errorMessage, successMessage: undefined});
        // Clear success timer on error
        if (this.successTimer) {
          clearTimeout(this.successTimer);
          this.successTimer = null;
        }
      });

    event.preventDefault();
  }

  render() {
    const batteryProfile = this.state.batteryProfile || { batteryType: '', batteryCapacity: '' };
    const chargingParameters = this.state.chargingParameters || {
      boostDuration: '', equalizationCycle: '', equalizationDuration: '',
      boostVoltage: '', boostReconnectChargingVoltage: '', floatVoltage: '',
      equalizationVoltage: '', chargingLimitVoltage: '', overVoltDisconnectVoltage: '',
      overVoltReconnectVoltage: '', lowVoltDisconnectVoltage: '', lowVoltReconnectVoltage: '',
      underVoltWarningVoltage: '', underVoltWarningReconnectVoltage: '', dischargingLimitVoltage: ''
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
                <Grid item xs={12} sm={6}>
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
                <Grid item xs={12} sm={6}>
                  <TextField
                    required
                    fullWidth
                    label="Battery Capacity (Ah)"
                    name="batteryCapacity"
                    value={batteryProfile.batteryCapacity}
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
                <Grid item xs={12} sm={6} md={4}>
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
                <Grid item xs={12} sm={6} md={4}>
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
                <Grid item xs={12} sm={6} md={4}>
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
                <Grid item xs={12} sm={6} md={4}>
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
                <Grid item xs={12} sm={6} md={4}>
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
                <Grid item xs={12} sm={6} md={4}>
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
                <Grid item xs={12} sm={6} md={4}>
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
                <Grid item xs={12} sm={6} md={4}>
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
                <Grid item xs={12} sm={6} md={3}>
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
                <Grid item xs={12} sm={6} md={3}>
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
                <Grid item xs={12} sm={6} md={3}>
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
                <Grid item xs={12} sm={6} md={3}>
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
                <Grid item xs={12} sm={6} md={4}>
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
                <Grid item xs={12} sm={6} md={4}>
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
                <Grid item xs={12} sm={6} md={4}>
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
};

export default Config
