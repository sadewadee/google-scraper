import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { AppShell } from "./components/Layout/AppShell"
import Dashboard from "./pages/Dashboard"
import Jobs from "./pages/Jobs"
import JobCreate from "./pages/Jobs/JobCreate"
import JobDetail from "./pages/Jobs/JobDetail"
import Workers from "./pages/Workers"
import Settings from "./pages/Settings"
import ProxyGate from "./pages/ProxyGate"

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AppShell>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/jobs" element={<Jobs />} />
            <Route path="/jobs/new" element={<JobCreate />} />
            <Route path="/jobs/:id" element={<JobDetail />} />
            <Route path="/workers" element={<Workers />} />
            <Route path="/proxies" element={<ProxyGate />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </AppShell>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export default App
