import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AuthProvider } from './auth/AuthContext'
import { RequireAuth } from './auth/RequireAuth'
import { Layout } from './components/Layout'
import { LoginPage } from './pages/LoginPage'
import { MembersPage } from './pages/MembersPage'
import { OrdersPage } from './pages/OrdersPage'
import { TemplatesPage } from './pages/TemplatesPage'
import { UploadPage } from './pages/UploadPage'
import { WpsRedirectPage } from './pages/WpsRedirectPage'

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/auth/wps" element={<WpsRedirectPage />} />
          <Route element={<RequireAuth />}>
            <Route element={<Layout />}>
              <Route path="/" element={<TemplatesPage />} />
              <Route path="/upload" element={<UploadPage />} />
              <Route path="/orders" element={<OrdersPage />} />
              <Route path="/members" element={<MembersPage />} />
            </Route>
          </Route>
          <Route path="*" element={<Navigate to="/login" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
