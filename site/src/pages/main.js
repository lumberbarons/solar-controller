import * as React from 'react';

import axios from 'axios';

import { Typography, Grid, Box } from '@material-ui/core';

import Metric from "../components/metric"

class Main extends React.Component {
  constructor(props) {
    super(props);

    this.state = {metrics: {}};
  }

  componentDidMount() {
    axios.get(`/api/metrics`)
      .then(res => {
        console.log(res.data);
        this.setState({metrics: res.data});
      }).catch(error => {
        console.log(JSON.stringify(error));
      });
  }
  
  render() {
    const metrics = this.state.metrics;

    return (
      <Box sx={{ width: '100%' }}>
        <Grid container spacing={2}>
          <Grid item xs={4}>
            <Metric title="Panel Power" value={metrics.arrayPower} unit="W" />
          </Grid>
          <Grid item xs={4}>
            <Metric title="Charging Power" value={metrics.chargingPower} unit="W" />
          </Grid>
          <Grid item xs={4}>
            <Metric title="Panel Voltage" value={metrics.arrayVoltage} unit="V" />
          </Grid>
          <Grid item xs={4}>
            <Metric title="Panel Current" value={metrics.arrayCurrent} unit="A" />
          </Grid>
          <Grid item xs={4}>
            <Metric title="Battery Voltage" value={metrics.batteryVoltage} unit="V" />
          </Grid>
          <Grid item xs={4}>
            <Metric title="Battery SOC" value={metrics.batterySoc} unit="%" />
          </Grid>
          <Grid item xs={4}>
            <Metric title="Device Temperature" value={metrics.deviceTemp} unit="C" />
          </Grid>
          <Grid item xs={4}>
            <Metric title="Battery Temperature" value={metrics.batteryTemp} unit="C" />
          </Grid>
          <Grid item xs={4}>
            <Metric title="Battery Voltage Minimum (Today)" value={metrics.batteryMinVoltage} unit="V" />
          </Grid>
          <Grid item xs={4}>
            <Metric title="Battery Voltage Maximum (Today)" value={metrics.batteryMaxVoltage} unit="V" />
          </Grid>
          <Grid item xs={4}>
            <Metric title="Energy Generated (Today)" value={metrics.energyGeneratedDaily} unit="KWh" />
          </Grid>
          <Grid item xs={4}>
            <Metric title="Energy Generated (Month)" value={metrics.energyGeneratedMonthly} unit="KWh" />
          </Grid>
        </Grid>
      </Box>
    );
  }
};

export default Main