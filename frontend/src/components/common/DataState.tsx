import { ErrorOutlineRounded, SearchOffRounded } from '@mui/icons-material'
import { Alert, Box, Button, CircularProgress, Typography } from '@mui/material'

export function LoadingState({ label = '正在读取证据…' }: { label?: string }) {
  return <Box className="data-state"><CircularProgress size={22} /><Typography>{label}</Typography></Box>
}

export function EmptyState({ title, detail, action }: { title: string; detail: string; action?: { label: string; onClick: () => void } }) {
  return <Box className="data-state empty-state"><SearchOffRounded /><Typography variant="h3">{title}</Typography><Typography>{detail}</Typography>{action && <Button variant="outlined" onClick={action.onClick}>{action.label}</Button>}</Box>
}

export function ErrorState({ message, retry }: { message: string; retry?: () => void }) {
  return <Alert severity="error" icon={<ErrorOutlineRounded />} action={retry && <Button color="inherit" size="small" onClick={retry}>重试</Button>}>{message}</Alert>
}

