import { Routes, Route } from 'react-router-dom';
import Landing from './pages/Landing';
import Dashboard from './pages/Dashboard';
import { ConnectButton } from '@rainbow-me/rainbowkit'; // ✅ Add this

export default function App() {
  return (
    <div className="min-h-screen bg-gray-100">
      <div className="flex justify-end p- shadow bg-white">
        <ConnectButton /> {/* ✅ Wallet connect button here */}
      </div>
      <br>
      </br>
      <Routes>
        <Route path="/" element={<Landing />} />
        <Route path="/dashboard" element={<Dashboard />} />
      </Routes>
    </div>
  );
}
