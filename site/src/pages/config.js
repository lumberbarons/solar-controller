import React from 'react';

import axios from 'axios';
import { Box, Grid, Container } from '@material-ui/core';
import { FormControl, InputLabel, Select, MenuItem, TextField, Button } from '@material-ui/core';

class Config extends React.Component {
  constructor(props) {
    super(props);

    this.state = {config: undefined};

    this.handleSubmit = this.handleSubmit.bind(this);
    this.handleInputChange = this.handleInputChange.bind(this);
  }

  componentDidMount() {
    axios.get(`/api/epever/config`)
      .then(res => {
        let clone = JSON.parse(JSON.stringify(res.data));
        this.setState({originalConfig: clone, config: res.data});
      }).catch(error => {
        console.error(JSON.stringify(error));
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

    axios.patch(`/api/epever/config`, payload)
      .then(res => {
        let clone = JSON.parse(JSON.stringify(res.data));
        this.setState({originalConfig: clone, config: res.data});
      }).catch(error => {
        this.setState({config: undefined,
          error: `Failed, status code: ${error.response.status}`});
      });

    event.preventDefault();
  }
  
  render() {
    if(this.state.config) {
      let config = this.state.config;

      return (
        <Container component="main" maxWidth="lg">
          <Box
            mt={2}
            component="form"
            autoComplete="off"
            onSubmit={this.handleSubmit}
          >
          <Grid container spacing={2}>
            <Grid item xs={4}>
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
            <Grid item xs={4}>
              <TextField
                required
                fullWidth
                label="Battery Capacity (Ah)"
                name="batteryCapacity"
                value={config.batteryCapacity}
                onChange={this.handleInputChange}
              />
            </Grid>
            <Grid item xs={4}>
              <TextField
                required
                fullWidth
                label="Time"
                name="time"
                value={config.time}
                onChange={this.handleInputChange}
              />
            </Grid>

            <Grid item xs={3}>
              <TextField
                required
                fullWidth
                label="Charging Limit Voltage"
                name="chargingLimitVoltage"
                value={config.chargingLimitVoltage}
                onChange={this.handleInputChange}
              />
            </Grid>
            <Grid item xs={3}>
              <TextField
                required
                fullWidth
                label="Boost Voltage"
                name="boostVoltage"
                value={config.boostVoltage}
                onChange={this.handleInputChange}
              />
            </Grid>
            <Grid item xs={3}>
              <TextField
                required
                fullWidth
                label="Boost Reconnect Voltage"
                name="boostReconnectChargingVoltage"
                value={config.boostReconnectChargingVoltage}
                onChange={this.handleInputChange}
              />
            </Grid>
            <Grid item xs={3}>
              <TextField
                required
                fullWidth
                label="Boost Duration"
                name="boostDuration"
                value={config.boostDuration}
                onChange={this.handleInputChange}
              />
            </Grid>

            <Grid item xs={3}>
              <TextField
                required
                fullWidth
                label="Float Voltage"
                name="floatVoltage"
                value={config.floatVoltage}
                onChange={this.handleInputChange}
              />
            </Grid>
            <Grid item xs={3}>
              <TextField
                required
                fullWidth
                label="Equalization Voltage"
                name="equalizationVoltage"
                value={config.equalizationVoltage}
                onChange={this.handleInputChange}
              />
            </Grid>
            <Grid item xs={3}>
              <TextField
                required
                fullWidth
                label="Equalization Cycle"
                name="equalizationCycle"
                value={config.equalizationCycle}
                onChange={this.handleInputChange}
              />
            </Grid>
            <Grid item xs={3}>
              <TextField
                required
                fullWidth
                label="Equalization Duration"
                name="equalizationDuration"
                value={config.equalizationDuration}
                onChange={this.handleInputChange}
              />
            </Grid>

            <Grid item xs={3}>
              <TextField
                required
                fullWidth
                label="Over Volt Disconnect"
                name="overVoltDisconnectVoltage"
                value={config.overVoltDisconnectVoltage}
                onChange={this.handleInputChange}
              />
            </Grid>
            <Grid item xs={3}>
              <TextField
                required
                fullWidth
                label="Over Volt Reconnect"
                name="overVoltReconnectVoltage"
                value={config.overVoltReconnectVoltage}
                onChange={this.handleInputChange}
              />
            </Grid>
            <Grid item xs={3}>
              <TextField
                required
                fullWidth
                label="Low Volt Disconnect"
                name="lowVoltDisconnectVoltage"
                value={config.lowVoltDisconnectVoltage}
                onChange={this.handleInputChange}
              />
            </Grid>
            <Grid item xs={3}>
              <TextField
                required
                fullWidth
                label="Low Volt Reconnect"
                name="lowVoltReconnectVoltage"
                value={config.lowVoltReconnectVoltage}
                onChange={this.handleInputChange}
              />
            </Grid>

            <Grid item xs={4}>
              <TextField
                required
                fullWidth
                label="Under Volt Warning"
                name="underVoltWarningVoltage"
                value={config.underVoltWarningVoltage}
                onChange={this.handleInputChange}
              />
            </Grid>
            <Grid item xs={4}>
              <TextField
                required
                fullWidth
                label="Under Volt Reconnect"
                name="underVoltWarningReconnectVoltage"
                value={config.underVoltWarningReconnectVoltage}
                onChange={this.handleInputChange}
              />
            </Grid>
            <Grid item xs={4}>
              <TextField
                required
                fullWidth
                label="Discharging Limit Voltage"
                name="dischargingLimitVoltage"
                value={config.dischargingLimitVoltage}
                onChange={this.handleInputChange}
              />
            </Grid>

            <Grid container justifyContent="flex-end">
              <Box mt={2}>
                <Button
                  type="submit"
                  variant="contained"
                  color="primary"
                >
                  Save
                </Button>
              </Box>
            </Grid>
          </Grid>
        </Box>
      </Container>
      );
    } else {
      return (
        <Box></Box>
      )
    }
  }
};

export default Config