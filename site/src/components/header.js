import React from 'react';

import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faSun } from '@fortawesome/free-solid-svg-icons'

import { Link } from "react-router-dom";
import { styled } from '@mui/material/styles';
import AppBar from '@mui/material/AppBar';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';

const Root = styled('div')({
  flexGrow: 1,
});

const Title = styled(Typography)({
  flexGrow: 1,
});

function Header() {

  return (
    <Root>
      <AppBar position="static">
        <Toolbar>
          <Title variant="h4">
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
          </Title>
          <Button component={Link} to="/config" color="inherit" size="large">Config</Button>
          <Button component={Link} to="/query" color="inherit" size="large">Query</Button>
        </Toolbar>
      </AppBar>
    </Root>
  );
}

export default Header;