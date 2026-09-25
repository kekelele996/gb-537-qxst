import { ArrowForwardRounded, LockOutlined, PersonOutlineRounded, ShieldOutlined } from '@mui/icons-material'
import { Alert, Box, Button, FormControl, InputLabel, MenuItem, Select, TextField, Typography } from '@mui/material'
import { FormEvent, useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'

const accounts = [
  { username: 'operator', password: 'operator123', label: 'PKI 操作员', detail: '维护、推演与状态推进' },
  { username: 'reviewer', password: 'reviewer123', label: '安全复核员', detail: '独立验证与审计查看' },
  { username: 'owner', password: 'owner123', label: '服务负责人', detail: '维护本团队依赖' },
  { username: 'auditor', password: 'auditor123', label: '审计员', detail: '只读证据审阅' },
  { username: 'admin', password: 'admin123', label: '系统管理员', detail: '完整管理权限' },
]

export function LoginPage() {
  const [username, setUsername] = useState('operator')
  const [password, setPassword] = useState('operator123')
  const { login, loading, error, clearError, token } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const target = (location.state as { from?: string } | null)?.from ?? '/rollovers'

  useEffect(() => { if (token) navigate(target, { replace: true }) }, [navigate, target, token])

  const chooseAccount = (value: string) => {
    const selected = accounts.find((account) => account.username === value) ?? accounts[0]
    setUsername(selected.username)
    setPassword(selected.password)
    clearError()
  }
  const submit = async (event: FormEvent) => {
    event.preventDefault()
    try { await login(username, password); navigate(target, { replace: true }) } catch { /* Store exposes the sanitized error. */ }
  }

  return <Box className="login-page">
    <Box className="login-identity">
      <Box className="login-brand"><Box className="brand-mark large"><span /><span /><span /></Box><Typography component="strong">CertRollover</Typography></Box>
      <Box className="identity-copy"><Typography className="eyebrow">PKI CHANGE EVIDENCE</Typography><Typography variant="h1">PKI 轮换推演台</Typography><Typography>在冻结快照上检查证书链、服务信任与轮换时间窗。证据留痕，双人复核，不触碰生产密钥。</Typography></Box>
      <Box className="identity-index"><Box><span>01</span><Typography>公钥材料离线校验</Typography></Box><Box><span>02</span><Typography>逐时间点依赖可达性</Typography></Box><Box><span>03</span><Typography>创建与复核职责分离</Typography></Box></Box>
      <Box className="boundary-stamp"><ShieldOutlined /><span>NO PRIVATE KEYS<br />OFFLINE DECISION SUPPORT</span></Box>
    </Box>
    <Box className="login-form-zone">
      <Box component="form" className="login-form" onSubmit={submit}>
        <Typography className="eyebrow">CONTROLLED ACCESS</Typography>
        <Typography variant="h2">进入推演工作台</Typography>
        <Typography className="login-lead">选择演示职责，或输入已配置的账号。</Typography>
        {error && <Alert severity="error" onClose={clearError}>{error}</Alert>}
        <FormControl fullWidth><InputLabel id="demo-account-label">演示职责</InputLabel><Select labelId="demo-account-label" label="演示职责" value={username} onChange={(event) => chooseAccount(event.target.value)}>{accounts.map((account) => <MenuItem key={account.username} value={account.username}><Box className="account-option"><strong>{account.label}</strong><span>{account.detail}</span></Box></MenuItem>)}</Select></FormControl>
        <TextField label="用户名" value={username} onChange={(event) => setUsername(event.target.value)} autoComplete="username" required slotProps={{ input: { startAdornment: <PersonOutlineRounded className="field-icon" /> } }} />
        <TextField label="密码" type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="current-password" required slotProps={{ input: { startAdornment: <LockOutlined className="field-icon" /> } }} />
        <Button type="submit" variant="contained" size="large" disabled={loading} endIcon={<ArrowForwardRounded />}>{loading ? '正在验证…' : '登录工作台'}</Button>
        <Typography className="login-footnote">本系统不能替代生产 PKI 变更评审与双人复核。</Typography>
      </Box>
    </Box>
  </Box>
}
