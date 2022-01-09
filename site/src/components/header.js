import React from 'react';
import PropTypes from "prop-types";
import { withRouter } from "react-router";

import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faSun } from '@fortawesome/free-solid-svg-icons'

import { Link } from "react-router-dom";
import { withStyles } from '@material-ui/core/styles';
import AppBar from '@material-ui/core/AppBar';
import Toolbar from '@material-ui/core/Toolbar';
import Typography from '@material-ui/core/Typography';
import Button from '@material-ui/core/Button';

const styles = theme => ({
  root: {
    flexGrow: 1,
  },
  title: {
    flexGrow: 1,
  }
});

class Header extends React.Component {
  constructor(props) {
    super(props);
  }
  
  render() {
    const { classes } = this.props;

    return (
      <div className={classes.root}>
        <AppBar position="static">
          <Toolbar>
            <Typography variant="h4" className={classes.title}>
              <Link to="/" style={{ color: 'inherit', textDecoration: 'inherit'}}>
                <FontAwesomeIcon 
                  icon={faSun} 
                  inverse 
                  style={{
                    marginRight: `0.5rem`,
                  }}
                />
                Solar Controller
              </Link>
            </Typography>
            <Button component={Link} to="/query" color="inherit" size="large">Query</Button>
          </Toolbar>
      </AppBar>
      </div>
    )
  }
}

Header.propTypes = {
  classes: PropTypes.object.isRequired,
  location: PropTypes.object.isRequired,
};

export default withRouter(withStyles(styles)(Header));