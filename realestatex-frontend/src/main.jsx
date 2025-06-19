// main.jsx or index.jsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'

import '@rainbow-me/rainbowkit/styles.css'
import { getDefaultConfig, RainbowKitProvider } from '@rainbow-me/rainbowkit'
import { WagmiProvider, http } from 'wagmi'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { sepolia } from 'wagmi/chains'
import { BrowserRouter } from 'react-router-dom'  // ✅ import BrowserRouter

const config = getDefaultConfig({
  appName: 'RealEstateX',
  projectId: '9e986855842776f609fbca8ff23cab3e',
  chains: [sepolia],
  transports: {
    [sepolia.id]: http("https://eth-sepolia.g.alchemy.com/v2/7PwkRZ0pPQoNUhIeCyoQA"),
  },
})

const queryClient = new QueryClient()

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <WagmiProvider config={config}>
        <RainbowKitProvider chains={[sepolia]}>
          <BrowserRouter> {/* ✅ wrap App in Router */}
            <App />
          </BrowserRouter>
        </RainbowKitProvider>
      </WagmiProvider>
    </QueryClientProvider>
  </React.StrictMode>
)
