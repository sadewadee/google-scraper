import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { ThemeProvider, CssBaseline } from "@mui/material"
import { Toaster } from "sonner"
import { theme } from "./theme"
import { Layout } from "./components/Layout"
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
    <ThemeProvider theme={theme}>
      <CssBaseline />
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
                  <Layout>
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
                  </Layout>
                </ProtectedRoute>
              }
            />
          </Routes>
        </BrowserRouter>
      </QueryClientProvider>
    </ThemeProvider>
  )
}

export default App
