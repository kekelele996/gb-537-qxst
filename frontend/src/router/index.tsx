import { Navigate, createBrowserRouter } from 'react-router-dom'
import { AppShell } from '../layout/AppShell'
import { AnchorsPage } from '../pages/AnchorsPage'
import { AuditPage } from '../pages/AuditPage'
import { ChainsPage } from '../pages/ChainsPage'
import { DependenciesPage } from '../pages/DependenciesPage'
import { LoginPage } from '../pages/LoginPage'
import { NotFoundPage } from '../pages/NotFoundPage'
import { RolloversPage } from '../pages/RolloversPage'
import { ProtectedRoute } from './ProtectedRoute'

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { element: <ProtectedRoute />, children: [{ element: <AppShell />, children: [
    { index: true, element: <Navigate to="/rollovers" replace /> },
    { path: '/anchors', element: <AnchorsPage /> },
    { path: '/chains', element: <ChainsPage /> },
    { path: '/dependencies', element: <DependenciesPage /> },
    { path: '/rollovers', element: <RolloversPage /> },
    { path: '/audit', element: <AuditPage /> },
  ] }] },
  { path: '*', element: <NotFoundPage /> },
])

