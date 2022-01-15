import React from 'react';

import axios from 'axios';
import { Box, Grid, Container, Card, CardContent, Typography } from '@material-ui/core';
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
        let resultDecimal = res.data.result;
        let resultHex = "0x" + resultDecimal.toString(16).padStart(4, '0');
        this.setState({resultHex: resultHex, resultDecimal: resultDecimal});
      }).catch(error => {
        console.log(error.response.data);
        console.log(error.response.status);
        console.log(error.response.headers);

        this.setState({resultHex: '', resultDecimal: 0, 
          error: `Failed, status code: ${error.response.status}`});
      });

    event.preventDefault();
  }
  
  render() {
    const register = this.state.register;
    const address = this.state.address;

    const resultHex = this.state.resultHex;
    const resultDecimal = this.state.resultDecimal;

    let results = ""
    if(resultHex !== '') {
      results = 
        <Grid container spacing={2}>
          <Grid item xs={12}>
            <Card>
                <CardContent>
                  <Typography sx={{ fontSize: 14 }} color="textSecondary" gutterBottom>
                      Result (Hex)
                  </Typography>
                  <Typography variant="h2" component="div">
                      {resultHex}
                  </Typography>
                </CardContent>
            </Card>
          </Grid>
        
          <Grid item xs={12}>
            <Card>
                <CardContent>
                  <Typography sx={{ fontSize: 14 }} color="textSecondary" gutterBottom>
                      Result (Decimal)
                  </Typography>
                  <Typography variant="h2" component="div">
                      {resultDecimal}
                  </Typography>
                </CardContent>
            </Card>
          </Grid>
        </Grid>
    } else if(this.state.error) {
      results = <Typography align="center" variant="h5" style={{ marginTop: "20px" }}>{this.state.error}</Typography>
    }

    return (
      <Container component="main" maxWidth="xs">
        <Box
          mt={2}
          component="form"
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

      <Box mt={2}>
        {results}
      </Box>
    </Container>
    );
  }
};

export default Query