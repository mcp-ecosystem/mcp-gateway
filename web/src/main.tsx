import {HeroUIProvider, ToastProvider} from "@heroui/react";
import React from "react";
import ReactDOM from "react-dom/client";

import App from "./App.tsx";
import './i18n';

import "./index.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <HeroUIProvider>
      <ToastProvider placement="bottom-right" />
      <main className="text-foreground bg-background h-screen overflow-hidden">
        <App />
      </main>
    </HeroUIProvider>
  </React.StrictMode>,
);
