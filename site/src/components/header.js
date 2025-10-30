import React from 'react';

import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faSun } from '@fortawesome/free-solid-svg-icons'

import { Link, useLocation } from "react-router-dom";
import { styled } from '@mui/material/styles';
import AppBar from '@mui/material/AppBar';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import Box from '@mui/material/Box';

const Root = styled('div')({
  flexGrow: 1,
});

const Title = styled(Typography)({
  flexGrow: 1,
  display: 'flex',
  alignItems: 'center',
});

function Header() {
  const location = useLocation();

  // Determine current tab based on path
  const getCurrentTab = () => {
    switch(location.pathname) {
      case '/':
        return 0;
      case '/config':
        return 1;
      default:
        return 0;
    }
  };

  return (
    <Root>
      <AppBar position="static">
        <Toolbar sx={{ minHeight: '64px' }}>
          <Title variant="h5">
            <Link to="/" style={{ color: 'inherit', textDecoration: 'inherit', display: 'flex', alignItems: 'center' }}>
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
          <Box sx={{ flexGrow: 0 }}>
            <Tabs
              value={getCurrentTab()}
              textColor="inherit"
              TabIndicatorProps={{
                style: {
                  backgroundColor: '#fff',
                  height: 3
                }
              }}
              sx={{
                '& .MuiTab-root': {
                  minHeight: '64px',
                  color: 'rgba(255, 255, 255, 0.7)',
                  fontWeight: 500,
                  fontSize: '0.9rem',
                  '&.Mui-selected': {
                    color: '#fff',
                    fontWeight: 600
                  },
                  '&:hover': {
                    color: '#fff',
                    backgroundColor: 'rgba(255, 255, 255, 0.1)'
                  }
                }
              }}
            >
              <Tab label="Dashboard" component={Link} to="/" />
              <Tab label="Config" component={Link} to="/config" />
            </Tabs>
          </Box>
        </Toolbar>
      </AppBar>
    </Root>
  );
}

export default Header;