import { AccountTreeRounded, AltRouteRounded, FactCheckRounded, LogoutRounded, MenuRounded, PolicyRounded, SecurityRounded } from '@mui/icons-material'
import { AppBar, Box, Divider, Drawer, IconButton, List, ListItemButton, ListItemIcon, ListItemText, Toolbar, Tooltip, Typography } from '@mui/material'
import { useState } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'
import { roleLabels } from '../utils/permissions'

const nav = [
  { to: '/anchors', label: '信任锚', eyebrow: 'ROOTS', icon: <SecurityRounded /> },
  { to: '/chains', label: '证书链', eyebrow: 'CHAINS', icon: <AccountTreeRounded /> },
  { to: '/dependencies', label: '服务依赖', eyebrow: 'REACH', icon: <AltRouteRounded /> },
  { to: '/rollovers', label: '轮换推演', eyebrow: 'SIMULATE', icon: <PolicyRounded /> },
  { to: '/audit', label: '审计中心', eyebrow: 'EVIDENCE', icon: <FactCheckRounded />, auditOnly: true },
]

function Brand() {
  return <Box className="brand"><Box className="brand-mark"><span /><span /><span /></Box><Box><Typography component="strong">CertRollover</Typography><Typography component="span">PKI 轮换推演台</Typography></Box></Box>
}

function Navigation({ close, showAudit }: { close?: () => void; showAudit: boolean }) {
  const location = useLocation()
  return <List className="nav-list">{nav.filter((item) => !item.auditOnly || showAudit).map((item) => <ListItemButton component={NavLink} to={item.to} key={item.to} selected={location.pathname.startsWith(item.to)} onClick={close}><ListItemIcon>{item.icon}</ListItemIcon><ListItemText primary={item.label} secondary={item.eyebrow} /></ListItemButton>)}</List>
}

export function AppShell() {
  const [mobileOpen, setMobileOpen] = useState(false)
  const { user, logout, can } = useAuth()
  const sidebar = <Box className="sidebar-inner"><Brand /><Navigation close={() => setMobileOpen(false)} showAudit={can('audit.read')} /><Box className="safety-boundary"><span>OFFLINE SIMULATION</span><Typography>仅处理公钥证书与冻结快照，不连接生产密钥系统。</Typography></Box><Divider /><Box className="user-block"><Box className="user-avatar">{user?.display_name?.slice(0, 1) ?? 'U'}</Box><Box><Typography component="strong">{user?.display_name}</Typography><Typography component="span">{user ? roleLabels[user.role] : ''} · {user?.team}</Typography></Box><Tooltip title="退出登录"><IconButton onClick={logout} aria-label="退出登录"><LogoutRounded fontSize="small" /></IconButton></Tooltip></Box></Box>
  return <Box className="app-frame">
    <AppBar className="mobile-appbar" position="fixed" elevation={0}><Toolbar><IconButton onClick={() => setMobileOpen(true)} aria-label="打开导航"><MenuRounded /></IconButton><Brand /></Toolbar></AppBar>
    <Box component="nav" className="desktop-sidebar">{sidebar}</Box>
    <Drawer open={mobileOpen} onClose={() => setMobileOpen(false)} slotProps={{ paper: { className: 'mobile-drawer' } }}>{sidebar}</Drawer>
    <Box component="main" className="main-content"><Outlet /></Box>
  </Box>
}
