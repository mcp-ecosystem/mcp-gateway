import React from 'react';
import { Toaster } from 'react-hot-toast';
import { BrowserRouter as Router, Routes, Route, Navigate, useLocation } from 'react-router-dom';

import { Layout } from './components/Layout';
import { LoginPage } from './pages/auth/login';
import { ChatInterface } from './pages/chat/chat-interface';
import { GatewayManager } from './pages/gateway/gateway-manager';

// Route guard component
function PrivateRoute({ children }: { children: React.ReactNode }) {
  const location = useLocation();
  const token = window.localStorage.getItem('token');

  if (!token) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  return <>{children}</>;
}

// Main layout component
function MainLayout() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<GatewayManager />} />
        <Route path="/chat" element={<ChatInterface />} />
        <Route path="/chat/:sessionId" element={<ChatInterface />} />
      </Routes>
    </Layout>
  );
}

export default function App() {
  return (
    <Router future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <Toaster position="top-right" />
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/*"
          element={
            <PrivateRoute>
              <MainLayout />
            </PrivateRoute>
          }
        />
      </Routes>
    </Router>
  );
}
