import {HeroUIProvider, ToastProvider} from "@heroui/react";
import { loader } from '@monaco-editor/react';
import React from "react";
import ReactDOM from "react-dom/client";
import axios from "axios";
import App from "./App.tsx";
import './i18n';

import "./index.css";

// Configure Monaco Editor to use local files
interface MonacoGlobal {
  MonacoEnvironment: {
    getWorkerUrl: (moduleId: string, label: string) => string;
  };
}

(globalThis as unknown as MonacoGlobal).MonacoEnvironment = {
  getWorkerUrl: function (_moduleId: string, _label: string) {
    // Use base worker for all cases since project only uses YAML
    return '/monaco-editor/vs/base/worker/workerMain.js';
  }
};

// Configure @monaco-editor/react to use local monaco-editor
loader.config({ 
  paths: { 
    vs: '/monaco-editor/vs' 
  } 
});

// Initialize monaco
loader.init().then(() => {
  // Monaco is now loaded and available
  console.log('Monaco Editor loaded from local files');
});

// Initialize theme immediately before React renders
const savedTheme = window.localStorage.getItem('theme');
if (savedTheme === 'dark') {
  document.documentElement.classList.add('dark');
} else if (savedTheme === 'light') {
  document.documentElement.classList.remove('dark');
}

// Fetch runtime config before rendering the app
const fetchRuntimeConfig = async () => {
  console.log("[RUNTIME_CONFIG] Fetching /api/runtime-config...");
  try {
    const response = await axios.get("/api/runtime-config");
    console.log("[RUNTIME_CONFIG] Fetched config:", response.data);
    window.RUNTIME_CONFIG = response.data;
  } catch (error) {
    console.error("[RUNTIME_CONFIG] Failed to load runtime config:", error);
    window.RUNTIME_CONFIG = {};
  }

  console.log("[RUNTIME_CONFIG] Rendering React app...");
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
};

fetchRuntimeConfig();

// Add global declaration for RUNTIME_CONFIG
declare global {
  interface Window {
    RUNTIME_CONFIG: any;
  }
}
