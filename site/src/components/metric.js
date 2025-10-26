import React from 'react';

import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';

class Metric extends React.Component {  
  render() {
    const title = this.props.title;
    const value = this.props.value;
    const unit = this.props.unit;

    return (
        <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                  {title}
              </Typography>
              <Typography variant="h5" component="div">
                  {value} {unit}
              </Typography>
            </CardContent>
        </Card>
    )
  }
}

export default Metric;