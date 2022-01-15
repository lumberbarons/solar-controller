import React from 'react';

import axios from 'axios';
import { Box, Grid, Container } from '@material-ui/core';
import { FormControl, InputLabel, Select, MenuItem, TextField, Button } from '@material-ui/core';

class Config extends React.Component {
  constructor(props) {
    super(props);

    this.state = {config: {batteryType: "unknown", batteryCapacity: 0, time: "",
                  boostVoltage: 0, equalizationVoltage: 0, floatVoltage: 0, 
                  boostReconnectVoltage: 0}};

    this.handleSubmit = this.handleSubmit.bind(this);
    this.handleInputChange = this.handleInputChange.bind(this);
  }

  componentDidMount() {
    axios.get(`/api/config`)
      .then(res => {
        console.log(res.data);
        this.setState({config: res.data});
      }).catch(error => {
        console.log(JSON.stringify(error));
      });
  }

  handleInputChange(event) {
    const value = event.target.value;
    const name = event.target.name;

    this.setState({
      [name]: value
    });
  }

  handleSubmit(event) {
    const payload = {};

    /* axios.patch(`/api/config`, payload)
      .then(res => {
        let resultDecimal = res.data.result;
        let resultHex = "0x" + resultDecimal.toString(16).padStart(4, '0');
        this.setState({resultHex: resultHex, resultDecimal: resultDecimal});
      }).catch(error => {
        console.log(error.response.data);
        console.log(error.response.status);
        console.log(error.response.headers);

        this.setState({resultHex: '', resultDecimal: 0, 
          error: `Failed, status code: ${error.response.status}`});
      }); */

    event.preventDefault();
  }
  
  render() {
    let batteryType = this.state.config.batteryType;
    let batteryCapacity = this.state.config.batteryCapacity;
    let time = this.state.config.time;

    let equalizationVoltage = this.state.config.equalizationVoltage;
    let boostVoltage = this.state.config.boostVoltage;
    let floatVoltage = this.state.config.floatVoltage;
    let boostReconnectVoltage = this.state.config.boostReconnectVoltage;

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

          <Grid item xs={3}>
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
              label="Float Voltage"
              name="floatVoltage"
              value={floatVoltage}
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