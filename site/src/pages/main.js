import * as React from 'react';

import axios from 'axios';

import { Grid, Box } from '@material-ui/core';

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

    let chargingStatus;
    switch(metrics.chargingStatus) {
      case 0:
        chargingStatus = "Not charging";
        break;
      case 1:
        chargingStatus = "Float";
        break;
      case 2:
        chargingStatus = "Boost";
        break;
      case 3:
        chargingStatus = "Equalization";
        break;
      default:
        chargingStatus = "Unknown";
        break;
    }

    return (
      <Box sx={{ width: '100%' }}>
        <Grid container spacing={2}>
          <Grid item sm={3} xs={6}>
            <Metric title="Panel Power" value={metrics.arrayPower} unit="W" />
          </Grid>
          <Grid item sm={3} xs={6}>
            <Metric title="Charging Power" value={metrics.chargingPower} unit="W" />
          </Grid>
          <Grid item sm={3} xs={6}>
            <Metric title="Charging Status" value={chargingStatus} unit="" />
          </Grid>
          <Grid item sm={3} xs={6}>
            <Metric title="Panel Voltage" value={metrics.arrayVoltage} unit="V" />
          </Grid>
          <Grid item sm={3} xs={6}>
            <Metric title="Panel Current" value={metrics.arrayCurrent} unit="A" />
          </Grid>
          <Grid item sm={3} xs={6}>
            <Metric title="Battery Voltage" value={metrics.batteryVoltage} unit="V" />
          </Grid>
          <Grid item sm={3} xs={6}>
            <Metric title="Battery SOC" value={metrics.batterySoc} unit="%" />
          </Grid>
          <Grid item sm={3} xs={6}>
            <Metric title="Device Temp" value={metrics.deviceTemp} unit="C" />
          </Grid>
          <Grid item sm={3} xs={6}>
            <Metric title="Battery Temp" value={metrics.batteryTemp} unit="C" />
          </Grid>
          <Grid item sm={3} xs={6}>
            <Metric title="Voltage Min" value={metrics.batteryMinVoltage} unit="V" />
          </Grid>
          <Grid item sm={3} xs={6}>
            <Metric title="Voltage Max" value={metrics.batteryMaxVoltage} unit="V" />
          </Grid>
          <Grid item sm={3} xs={6}>
            <Metric title="Generated (Today)" value={metrics.energyGeneratedDaily} unit="KWh" />
          </Grid>
          <Grid item sm={3} xs={6}>
            <Metric title="Generated (Month)" value={metrics.energyGeneratedMonthly} unit="KWh" />
          </Grid>
        </Grid>
      </Box>
    );
  }
};

export default Main