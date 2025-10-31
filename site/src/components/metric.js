import React from 'react';
import PropTypes from 'prop-types';

import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';

class Metric extends React.Component {
  render() {
    const title = this.props.title;
    const value = this.props.value;
    const unit = this.props.unit;

    return (
        <Card
          sx={{
            height: '100%',
            minHeight: 120,
            display: 'flex',
            flexDirection: 'column',
            background: 'linear-gradient(135deg, #2e7d32 0%, #1b5e20 100%)',
            color: 'white',
            boxShadow: '0 4px 20px rgba(0,0,0,0.1)',
            transition: 'transform 0.2s, box-shadow 0.2s',
            '&:hover': {
              transform: 'translateY(-4px)',
              boxShadow: '0 8px 30px rgba(0,0,0,0.15)'
            }
          }}
        >
            <CardContent sx={{ flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
              <Typography
                sx={{
                  color: 'rgba(255,255,255,0.85)',
                  fontSize: '0.875rem',
                  fontWeight: 500,
                  letterSpacing: '0.5px',
                  textTransform: 'uppercase'
                }}
                gutterBottom
              >
                  {title}
              </Typography>
              <Typography
                variant="h4"
                component="div"
                sx={{
                  fontWeight: 700,
                  color: 'white'
                }}
              >
                  {value} {unit}
              </Typography>
            </CardContent>
        </Card>
    )
  }
}

Metric.propTypes = {
  title: PropTypes.string.isRequired,
  value: PropTypes.oneOfType([PropTypes.string, PropTypes.number]).isRequired,
  unit: PropTypes.string
};

export default Metric;