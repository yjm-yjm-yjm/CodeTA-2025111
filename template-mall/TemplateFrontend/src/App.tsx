import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AuthProvider } from './auth/AuthContext'
import { Layout } from './components/Layout'
import { LoginPage } from './pages/LoginPage'
import { MembershipPage } from './pages/MembershipPage'
import { OrdersPage } from './pages/OrdersPage'
import { TemplatesPage } from './pages/TemplatesPage'

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<Layout />}>
            <Route path="/" element={<TemplatesPage />} />
            <Route path="/membership" element={<MembershipPage />} />
            <Route path="/orders" element={<OrdersPage />} />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
