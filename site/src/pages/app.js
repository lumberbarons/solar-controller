import React from "react";
import { Switch, Route } from "react-router-dom";
import { withRouter } from "react-router";
import PropTypes from "prop-types"

import Header from "../components/header";
import Main from "./main";
import Query from "./query";

class App extends React.Component {
  constructor(props) {
    super(props);
  }

  render() {
    return (
      <div>
        <Header />
        <div
          style={{
            margin: `0 auto`,
            padding: `1.0rem 1.0875rem 1.0rem`,
          }}
        >
          <main>
            <Switch>
              <Route path="/query">
                <Query />
              </Route>
              <Route path="/">
                <Main />
              </Route>
          </Switch>
          </main>
        </div>
      </div>
    );
  }
}

App.propTypes = {
  history: PropTypes.object.isRequired,
  location: PropTypes.object.isRequired,
};

export default withRouter(App)