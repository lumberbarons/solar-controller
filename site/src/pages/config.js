import React from 'react';

import axios from 'axios';
import { Box, Grid, Container } from '@material-ui/core';
import { FormControl, InputLabel, Select, MenuItem, TextField, Button } from '@material-ui/core';

class Config extends React.Component {
  constructor(props) {
    super(props);

    this.state = {config: this.getEmptyConfig()};

    this.handleSubmit = this.handleSubmit.bind(this);
    this.handleInputChange = this.handleInputChange.bind(this);
  }

  componentDidMount() {
    axios.get(`/api/config`)
      .then(res => {
        let clone = JSON.parse(JSON.stringify(res.data));
        this.setState({originalConfig: clone, config: res.data});
      }).catch(error => {
        console.error(JSON.stringify(error));
      });
  }

  getEmptyConfig() {
    return {batteryType: "unknown", batteryCapacity: 0, time: "",
        boostVoltage: 0, equalizationVoltage: 0, equalizationCycle: 0, equalizationDuration: 0,
        floatVoltage: 0, boostReconnectVoltage: 0, boostDuration: 0};
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

    if(config.equalizationCycle !== originalConfig.equalizationCycle) {
      payload.equalizationCycle = parseInt(config.equalizationCycle);
    }

    if(config.boostVoltage !== originalConfig.boostVoltage) {
      payload.boostVoltage = parseFloat(config.boostVoltage);
    }

    if(config.floatVoltage !== originalConfig.floatVoltage) {
      payload.floatVoltage = parseFloat(config.floatVoltage);
    }

    axios.patch(`/api/config`, payload)
      .then(res => {
        let clone = JSON.parse(JSON.stringify(res.data));
        this.setState({originalConfig: clone, config: res.data});
      }).catch(error => {
        this.setState({config: this.getEmptyConfig(),
          error: `Failed, status code: ${error.response.status}`});
      });

    event.preventDefault();
  }
  
  render() {
    let batteryType = this.state.config.batteryType;
    let batteryCapacity = this.state.config.batteryCapacity;
    let time = this.state.config.time;

    let floatVoltage = this.state.config.floatVoltage;

    let equalizationVoltage = this.state.config.equalizationVoltage;
    let equalizationCycle = this.state.config.equalizationCycle;
    let equalizationDuration = this.state.config.equalizationDuration;
    
    let boostVoltage = this.state.config.boostVoltage;
    let boostReconnectVoltage = this.state.config.boostReconnectVoltage;
    let boostDuration = this.state.config.boostDuration;

    return (
      <Container component="main" maxWidth="md">
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
                value={batteryType}
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
              id="outlined-required"
              label="Battery Capacity"
              name="batteryCapacity"
              value={batteryCapacity}
              onChange={this.handleInputChange}
            />
          </Grid>

          <Grid item xs={4}>
            <TextField
              required
              fullWidth
              id="outlined-required"
              label="Time"
              name="time"
              value={time}
              onChange={this.handleInputChange}
            />
          </Grid>

          <Grid item xs={4}>
            <TextField
              required
              fullWidth
              id="outlined-required"
              label="Equalization Voltage"
              name="equalizationVoltage"
              value={equalizationVoltage}
              onChange={this.handleInputChange}
            />
          </Grid>

          <Grid item xs={4}>
            <TextField
              required
              fullWidth
              id="outlined-required"
              label="Equalization Cycle"
              name="equalizationCycle"
              value={equalizationCycle}
              onChange={this.handleInputChange}
            />
          </Grid>

          <Grid item xs={4}>
            <TextField
              required
              fullWidth
              id="outlined-required"
              label="Equalization Duration"
              name="equalizationDuration"
              value={equalizationDuration}
              onChange={this.handleInputChange}
            />
          </Grid>

          <Grid item xs={3}>
            <TextField
              required
              fullWidth
              id="outlined-required"
              label="Boost Voltage"
              name="boostVoltage"
              value={boostVoltage}
              onChange={this.handleInputChange}
            />
          </Grid>

          <Grid item xs={3}>
            <TextField
              required
              fullWidth
              id="outlined-required"
              label="Boost Reconnect Voltage"
              name="boostReconnectVoltage"
              value={boostReconnectVoltage}
              onChange={this.handleInputChange}
            />
          </Grid>

          <Grid item xs={3}>
            <TextField
              required
              fullWidth
              id="outlined-required"
              label="Boost Duration"
              name="boostDuration"
              value={boostDuration}
              onChange={this.handleInputChange}
            />
          </Grid>
          
          <Grid item xs={3}>
            <TextField
              required
              fullWidth
              id="outlined-required"
              label="Float Voltage"
              name="floatVoltage"
              value={floatVoltage}
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
  }
};

export default Config