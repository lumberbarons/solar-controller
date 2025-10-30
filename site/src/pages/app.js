import React from "react";
import { Routes, Route } from "react-router-dom";

import Header from "../components/header";
import Main from "./main";
import Query from "./query";
import Config from "./config";

function App() {
  return (
    <div style={{ width: '100%', overflow: 'hidden' }}>
      <Header />
      <main>
        <Routes>
          <Route path="/query" element={<Query />} />
          <Route path="/config" element={<Config />} />
          <Route path="/" element={<Main />} />
        </Routes>
      </main>
    </div>
  );
}

export default App;