import React from 'react';

import axios from 'axios';
import { Box, Grid, Container, Alert, Typography, Paper, IconButton } from '@mui/material';
import { FormControl, InputLabel, Select, MenuItem, TextField, Button } from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';

class Config extends React.Component {
  constructor(props) {
    super(props);

    this.state = {config: undefined, error: undefined, loading: false};

    this.handleSubmit = this.handleSubmit.bind(this);
    this.handleInputChange = this.handleInputChange.bind(this);
    this.fetchConfig = this.fetchConfig.bind(this);
  }

  componentDidMount() {
    this.fetchConfig();
  }

  fetchConfig() {
    this.setState({loading: true});

    axios.get(`/api/solar/config`)
      .then(res => {
        let clone = JSON.parse(JSON.stringify(res.data));
        this.setState({originalConfig: clone, config: res.data, error: undefined, loading: false});
      }).catch(error => {
        console.error(JSON.stringify(error));
        const errorMessage = error.response
          ? `Failed to load configuration: ${error.response.status} ${error.response.statusText}`
          : `Failed to load configuration: ${error.message}`;
        this.setState({config: undefined, error: errorMessage, loading: false});
      });
  }

  handleInputChange(event) {
    const value = event.target.value;
    const name = event.target.name;

    let config = this.state.config;
    config[name] = value;

    this.setState({
      config: config
    });
  }

  handleSubmit(event) {
    const payload = {};

    const originalConfig = this.state.originalConfig;
    const config = this.state.config;

    if(config.batteryType !== originalConfig.batteryType) {
      payload.batteryType = config.batteryType;
    }

    if(config.equalizationCycle !== originalConfig.equalizationCycle) {
      payload.equalizationCycle = parseInt(config.equalizationCycle);
    }

    if(config.equalizationVoltage !== originalConfig.equalizationVoltage) {
      payload.equalizationVoltage = parseFloat(config.equalizationVoltage);
    }

    if(config.equalizationDuration !== originalConfig.equalizationDuration) {
      payload.equalizationDuration = parseFloat(config.equalizationDuration);
    }

    if(config.boostVoltage !== originalConfig.boostVoltage) {
      payload.boostVoltage = parseFloat(config.boostVoltage);
    }

    if(config.boostDuration !== originalConfig.boostDuration) {
      payload.boostDuration = parseInt(config.boostDuration);
    }

    if(config.floatVoltage !== originalConfig.floatVoltage) {
      payload.floatVoltage = parseFloat(config.floatVoltage);
    }

    if(config.chargingLimitVoltage !== originalConfig.chargingLimitVoltage) {
      payload.chargingLimitVoltage = parseFloat(config.chargingLimitVoltage);
    }

    axios.patch(`/api/solar/config`, payload)
      .then(res => {
        let clone = JSON.parse(JSON.stringify(res.data));
        this.setState({originalConfig: clone, config: res.data, error: undefined});
      }).catch(error => {
        const errorMessage = error.response
          ? `Failed to save configuration: ${error.response.status} ${error.response.statusText}`
          : `Failed to save configuration: ${error.message}`;
        this.setState({error: errorMessage});
      });

    event.preventDefault();
  }
  
  render() {
    if(this.state.error) {
      return (
        <Box sx={{ width: '100%', p: 2, backgroundColor: 'white', minHeight: '100vh', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
          <Box sx={{
            mt: 4,
            p: 3,
            backgroundColor: '#ffebee',
            border: '2px solid #c62828',
            borderRadius: 2,
            boxShadow: '0 4px 12px rgba(0,0,0,0.1)',
            maxWidth: 600,
            width: '100%'
          }}>
            <Typography variant="h6" sx={{ color: '#c62828', fontWeight: 600, mb: 1 }}>
              Error Loading Configuration
            </Typography>
            <Typography sx={{ color: '#d32f2f' }}>
              {this.state.error}
            </Typography>
          </Box>
          <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
            <IconButton
              onClick={this.fetchConfig}
              disabled={this.state.loading}
              sx={{
                backgroundColor: 'white',
                boxShadow: '0 4px 12px rgba(0,0,0,0.1)',
                '&:hover': {
                  backgroundColor: '#2e7d32',
                  color: 'white',
                  transform: 'scale(1.1)'
                },
                transition: 'all 0.2s'
              }}
              aria-label="refresh configuration"
            >
              <RefreshIcon />
            </IconButton>
          </Box>
        </Box>
      );
    }

    if(this.state.config) {
      let config = this.state.config;

      return (
        <Box sx={{ width: '100%', p: 2, backgroundColor: 'white', minHeight: '100vh', maxWidth: 1400, mx: 'auto' }}>
          <Box
            component="form"
            autoComplete="off"
            onSubmit={this.handleSubmit}
          >
            {/* Battery Settings */}
            <Typography variant="h6" sx={{ mb: 1.5, mt: 1, fontWeight: 600, color: '#1b5e20' }}>
              Battery Settings
            </Typography>
            <Paper elevation={2} sx={{ p: 2, mb: 2, borderRadius: 2 }}>
              <Grid container spacing={2}>
                <Grid item xs={12} sm={6}>
                  <FormControl fullWidth>
                    <InputLabel>Battery Type</InputLabel>
                    <Select
                      name="batteryType"
                      value={config.batteryType}
                      label="Battery Type"
                      onChange={this.handleInputChange}
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
                    value={config.batteryCapacity}
                    onChange={this.handleInputChange}
                  />
                </Grid>
              </Grid>
            </Paper>

            {/* Charging Voltage Settings */}
            <Typography variant="h6" sx={{ mb: 1.5, fontWeight: 600, color: '#1b5e20' }}>
              Charging Voltage Settings
            </Typography>
            <Paper elevation={2} sx={{ p: 2, mb: 2, borderRadius: 2 }}>
              <Grid container spacing={2}>
                <Grid item xs={12} sm={6} md={4}>
                  <TextField
                    required
                    fullWidth
                    label="Charging Limit Voltage"
                    name="chargingLimitVoltage"
                    value={config.chargingLimitVoltage}
                    onChange={this.handleInputChange}
                  />
                </Grid>
                <Grid item xs={12} sm={6} md={4}>
                  <TextField
                    required
                    fullWidth
                    label="Boost Voltage"
                    name="boostVoltage"
                    value={config.boostVoltage}
                    onChange={this.handleInputChange}
                  />
                </Grid>
                <Grid item xs={12} sm={6} md={4}>
                  <TextField
                    required
                    fullWidth
                    label="Boost Reconnect Voltage"
                    name="boostReconnectChargingVoltage"
                    value={config.boostReconnectChargingVoltage}
                    onChange={this.handleInputChange}
                  />
                </Grid>
                <Grid item xs={12} sm={6} md={4}>
                  <TextField
                    required
                    fullWidth
                    label="Boost Duration (min)"
                    name="boostDuration"
                    value={config.boostDuration}
                    onChange={this.handleInputChange}
                  />
                </Grid>
                <Grid item xs={12} sm={6} md={4}>
                  <TextField
                    required
                    fullWidth
                    label="Float Voltage"
                    name="floatVoltage"
                    value={config.floatVoltage}
                    onChange={this.handleInputChange}
                  />
                </Grid>
                <Grid item xs={12} sm={6} md={4}>
                  <TextField
                    required
                    fullWidth
                    label="Equalization Voltage"
                    name="equalizationVoltage"
                    value={config.equalizationVoltage}
                    onChange={this.handleInputChange}
                  />
                </Grid>
                <Grid item xs={12} sm={6} md={4}>
                  <TextField
                    required
                    fullWidth
                    label="Equalization Cycle (days)"
                    name="equalizationCycle"
                    value={config.equalizationCycle}
                    onChange={this.handleInputChange}
                  />
                </Grid>
                <Grid item xs={12} sm={6} md={4}>
                  <TextField
                    required
                    fullWidth
                    label="Equalization Duration (min)"
                    name="equalizationDuration"
                    value={config.equalizationDuration}
                    onChange={this.handleInputChange}
                  />
                </Grid>
              </Grid>
            </Paper>

            {/* Protection Voltage Settings */}
            <Typography variant="h6" sx={{ mb: 1.5, fontWeight: 600, color: '#1b5e20' }}>
              Protection Voltage Settings
            </Typography>
            <Paper elevation={2} sx={{ p: 2, mb: 2, borderRadius: 2 }}>
              <Grid container spacing={2}>
                <Grid item xs={12} sm={6} md={3}>
                  <TextField
                    required
                    fullWidth
                    label="Over Volt Disconnect"
                    name="overVoltDisconnectVoltage"
                    value={config.overVoltDisconnectVoltage}
                    onChange={this.handleInputChange}
                  />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                  <TextField
                    required
                    fullWidth
                    label="Over Volt Reconnect"
                    name="overVoltReconnectVoltage"
                    value={config.overVoltReconnectVoltage}
                    onChange={this.handleInputChange}
                  />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                  <TextField
                    required
                    fullWidth
                    label="Low Volt Disconnect"
                    name="lowVoltDisconnectVoltage"
                    value={config.lowVoltDisconnectVoltage}
                    onChange={this.handleInputChange}
                  />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                  <TextField
                    required
                    fullWidth
                    label="Low Volt Reconnect"
                    name="lowVoltReconnectVoltage"
                    value={config.lowVoltReconnectVoltage}
                    onChange={this.handleInputChange}
                  />
                </Grid>
                <Grid item xs={12} sm={6} md={4}>
                  <TextField
                    required
                    fullWidth
                    label="Under Volt Warning"
                    name="underVoltWarningVoltage"
                    value={config.underVoltWarningVoltage}
                    onChange={this.handleInputChange}
                  />
                </Grid>
                <Grid item xs={12} sm={6} md={4}>
                  <TextField
                    required
                    fullWidth
                    label="Under Volt Reconnect"
                    name="underVoltWarningReconnectVoltage"
                    value={config.underVoltWarningReconnectVoltage}
                    onChange={this.handleInputChange}
                  />
                </Grid>
                <Grid item xs={12} sm={6} md={4}>
                  <TextField
                    required
                    fullWidth
                    label="Discharging Limit Voltage"
                    name="dischargingLimitVoltage"
                    value={config.dischargingLimitVoltage}
                    onChange={this.handleInputChange}
                  />
                </Grid>
              </Grid>
            </Paper>

            <Box sx={{ display: 'flex', justifyContent: 'flex-end', mt: 2 }}>
              <Button
                type="submit"
                variant="contained"
                color="primary"
                size="large"
                sx={{
                  backgroundColor: '#2e7d32',
                  '&:hover': {
                    backgroundColor: '#1b5e20'
                  },
                  px: 4,
                  py: 1.5,
                  fontWeight: 600
                }}
              >
                Save Configuration
              </Button>
            </Box>
          </Box>
        </Box>
      );
    } else {
      return (
        <Box></Box>
      )
    }
  }
};

export default Config