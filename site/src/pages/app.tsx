import { useState, useEffect } from "react";
import { Routes, Route } from "react-router-dom";
import axios from "axios";

import Header from "../components/header";
import Main from "./main";
import Config from "./config";
import type { VersionInfo } from "../api/types";

function App() {
  const [version, setVersion] = useState<VersionInfo | null>(null);

  useEffect(() => {
    // Fetch version info from the backend
    axios.get<VersionInfo>('/api/info')
      .then(response => {
        setVersion(response.data);
      })
      .catch(error => {
        console.error('Failed to fetch version info:', error);
      });
  }, []);

  return (
    <div style={{ width: '100%', overflow: 'hidden' }}>
      <Header version={version} />
      <main>
        <Routes>
          <Route path="/config" element={<Config />} />
          <Route path="/" element={<Main />} />
        </Routes>
      </main>
    </div>
  );
}

export default App;
