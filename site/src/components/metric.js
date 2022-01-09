import React from 'react';

import Card from '@material-ui/core/Card';
import CardContent from '@material-ui/core/CardContent';
import Typography from '@material-ui/core/Typography';

class Metric extends React.Component {  
  render() {
    const title = this.props.title;
    const value = this.props.value;
    const unit = this.props.unit;

    return (
        <Card>
            <CardContent>
            <Typography sx={{ fontSize: 14 }} color="textSecondary" gutterBottom>
                {title}
            </Typography>
            <Typography variant="h2" component="div">
                {value} {unit}
            </Typography>
            </CardContent>
        </Card>
    )
  }
}

export default Metric;