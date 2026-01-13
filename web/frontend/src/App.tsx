import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { Toaster } from "sonner"
import { AppShell } from "./components/Layout/AppShell"
import { isAuthenticated } from "./api/client"
import Dashboard from "./pages/Dashboard"
import Jobs from "./pages/Jobs"
import JobCreate from "./pages/Jobs/JobCreate"
import JobDetail from "./pages/Jobs/JobDetail"
import Workers from "./pages/Workers"
import Settings from "./pages/Settings"
import ProxyGate from "./pages/ProxyGate"
import Results from "./pages/Results"
import Login from "./pages/Login"

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

// Protected Route wrapper
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Toaster position="top-right" richColors closeButton />
      <BrowserRouter>
        <Routes>
          {/* Public route */}
          <Route path="/login" element={<Login />} />

          {/* Protected routes */}
          <Route
            path="/*"
            element={
              <ProtectedRoute>
                <AppShell>
                  <Routes>
                    <Route path="/" element={<Dashboard />} />
                    <Route path="/jobs" element={<Jobs />} />
                    <Route path="/jobs/new" element={<JobCreate />} />
                    <Route path="/jobs/:id" element={<JobDetail />} />
                    <Route path="/results" element={<Results />} />
                    <Route path="/workers" element={<Workers />} />
                    <Route path="/proxies" element={<ProxyGate />} />
                    <Route path="/settings" element={<Settings />} />
                    <Route path="*" element={<Navigate to="/" replace />} />
                  </Routes>
                </AppShell>
              </ProtectedRoute>
            }
          />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export default App
