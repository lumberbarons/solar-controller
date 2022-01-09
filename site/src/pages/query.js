import React from 'react';

import axios from 'axios';
import { Box, Grid, Container } from '@material-ui/core';
import { FormControl, InputLabel, Select, MenuItem, TextField, Button } from '@material-ui/core';

class Query extends React.Component {
  constructor(props) {
    super(props);

    this.state = {register: '4', address: '0x3100', resultHex: '', resultDecimal: 0};

    this.handleSubmit = this.handleSubmit.bind(this);
    this.handleInputChange = this.handleInputChange.bind(this);
  }

  handleInputChange(event) {
    const value = event.target.value;
    const name = event.target.name;

    this.setState({
      [name]: value
    });
  }

  handleSubmit(event) {
    const payload = {register: parseInt(this.state.register), address: this.state.address};
    axios.post(`/api/query`, payload)
      .then(res => {
        console.log(res.data);
      }).catch(error => {
        console.log(JSON.stringify(error));
      });

    event.preventDefault();
  }
  
  render() {
    const register = this.state.register;
    const address = this.state.address;

    return (
      <Container component="main" maxWidth="xs">
      <Box
        component="form"
        noValidate
        autoComplete="off"
        onSubmit={this.handleSubmit}
      >
      <Grid container spacing={2}>
        <Grid item xs={12}>
          <FormControl fullWidth>
            <InputLabel>Register</InputLabel>
            <Select
              name="register"
              value={register}
              label="Register"
              onChange={this.handleInputChange}
            >
              <MenuItem value="1">Coils</MenuItem>
              <MenuItem value="2">Discrete</MenuItem>
              <MenuItem value="3">Holding</MenuItem>
              <MenuItem value="4">Input</MenuItem>
            </Select>
          </FormControl>
        </Grid>

        <Grid item xs={12}>
          <TextField
            required
            fullWidth
            id="outlined-required"
            label="Address"
            name="address"
            value={address}
            onChange={this.handleInputChange}
          />
        </Grid>
        <Grid item xs={12}>
          <Button
            type="submit"
            fullWidth
            variant="contained"
            color="primary"
          >
            Query
          </Button>
        </Grid>
      </Grid>
    </Box>
    </Container>
    );
  }
};

export default Query