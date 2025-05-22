import React from 'react';
import { Routes, Route } from 'react-router-dom';
import Sidebar from './components/Sidebar';
import Topbar from './components/Topbar';
import Dashboard from './pages/Dashboard';
import Devices from './pages/Devices';
import Settings from './pages/Settings';
import ColorPalette from './pages/ColorPalette';
import './App.css';

export default function App() {
  return (
    
      <div className="flex flex-col h-screen">
        <Topbar />
        <div className="flex flex-1 bg-primary overflow-hidden">
          <Sidebar />
            <main className="flex-1 overflow-auto rounded-tl-xl bg-base-200">
              <Routes>
                <Route path="/" element={<Dashboard />} />
                <Route path="/devices" element={<Devices />} />
                <Route path="/settings" element={<Settings />} />
                <Route path="/color" element={<ColorPalette />} />
              </Routes>
            </main>
        </div>
    </div>
  );
}