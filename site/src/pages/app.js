import React from "react";
import { Routes, Route } from "react-router-dom";

import Header from "../components/header";
import Main from "./main";
import Query from "./query";
import Config from "./config";

function App() {
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
          <Routes>
            <Route path="/query" element={<Query />} />
            <Route path="/config" element={<Config />} />
            <Route path="/" element={<Main />} />
          </Routes>
        </main>
      </div>
    </div>
  );
}

export default App;