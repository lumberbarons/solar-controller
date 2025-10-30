import * as React from 'react';

import axios from 'axios';

import { Grid, Box, Container, Alert, IconButton, Typography } from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';

import Metric from "../components/metric"

class Main extends React.Component {
  constructor(props) {
    super(props);

    this.state = {metrics: {}, error: undefined, loading: false};

    this.fetchMetrics = this.fetchMetrics.bind(this);
  }

  componentDidMount() {
    this.fetchMetrics();
  }

  fetchMetrics() {
    this.setState({loading: true});

    axios.get(`/api/solar/metrics`)
      .then(res => {
        this.setState({metrics: res.data, error: undefined, loading: false});
      }).catch(error => {
        console.error(JSON.stringify(error));
        const errorMessage = error.response
          ? `Failed to load metrics: ${error.response.status} ${error.response.statusText}`
          : `Failed to load metrics: ${error.message}`;
        this.setState({metrics: {}, error: errorMessage, loading: false});
      });
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
              Error Loading Metrics
            </Typography>
            <Typography sx={{ color: '#d32f2f' }}>
              {this.state.error}
            </Typography>
          </Box>
          <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
            <IconButton
              onClick={this.fetchMetrics}
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
              aria-label="refresh metrics"
            >
              <RefreshIcon />
            </IconButton>
          </Box>
        </Box>
      );
    }

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
      <>
        <Box sx={{ width: '100%', p: 2, backgroundColor: 'white', minHeight: '100vh' }}>
          {/* Solar Panel Metrics */}
          <Typography variant="h6" sx={{ mb: 1, mt: 1, fontWeight: 600, color: '#1b5e20' }}>
            Solar Panel
          </Typography>
          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid item sm={3} xs={6}>
              <Metric title="Panel Power" value={metrics.arrayPower} unit="W" />
            </Grid>
            <Grid item sm={3} xs={6}>
              <Metric title="Panel Voltage" value={metrics.arrayVoltage} unit="V" />
            </Grid>
            <Grid item sm={3} xs={6}>
              <Metric title="Panel Current" value={metrics.arrayCurrent} unit="A" />
            </Grid>
            <Grid item sm={3} xs={6}>
              <Metric title="Generated (Today)" value={metrics.energyGeneratedDaily} unit="KWh" />
            </Grid>
          </Grid>

          {/* Charging Metrics */}
          <Typography variant="h6" sx={{ mb: 1, fontWeight: 600, color: '#1b5e20' }}>
            Charging
          </Typography>
          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid item sm={3} xs={6}>
              <Metric title="Charging Power" value={metrics.chargingPower} unit="W" />
            </Grid>
            <Grid item sm={3} xs={6}>
              <Metric title="Charging Current" value={metrics.chargingCurrent} unit="A" />
            </Grid>
            <Grid item sm={3} xs={6}>
              <Metric title="Charging Status" value={chargingStatus} unit="" />
            </Grid>
          </Grid>

          {/* Battery Metrics */}
          <Typography variant="h6" sx={{ mb: 1, fontWeight: 600, color: '#1b5e20' }}>
            Battery
          </Typography>
          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid item sm={3} xs={6}>
              <Metric title="Battery Voltage" value={metrics.batteryVoltage} unit="V" />
            </Grid>
            <Grid item sm={3} xs={6}>
              <Metric title="Battery SOC" value={metrics.batterySoc} unit="%" />
            </Grid>
          </Grid>

          <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
            <IconButton
              onClick={this.fetchMetrics}
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
              aria-label="refresh metrics"
            >
              <RefreshIcon />
            </IconButton>
          </Box>
        </Box>
      </>
    );
  }
}

export default Main