import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { SWRConfig } from "swr";
import { Toast } from "@heroui/react";

import { App } from "./app/App";
import { ThemeProvider } from "./app/ThemeProvider";
import { swrConfig } from "./lib/swr";
import { applyTheme, readTheme } from "./lib/theme";
import "./index.css";

applyTheme(readTheme());

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider>
      <SWRConfig value={swrConfig}>
        <BrowserRouter>
          <App />
        </BrowserRouter>
        <Toast.Provider placement="bottom end" />
      </SWRConfig>
    </ThemeProvider>
  </StrictMode>,
);
