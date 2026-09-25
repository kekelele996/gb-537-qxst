import type { ReactNode } from 'react'
import { CloseRounded } from '@mui/icons-material'
import { Box, Drawer, IconButton, Typography } from '@mui/material'

export function FormDrawer({ open, onClose, title, eyebrow, children }: { open: boolean; onClose: () => void; title: string; eyebrow: string; children: ReactNode }) {
  return <Drawer anchor="right" open={open} onClose={onClose} slotProps={{ paper: { className: 'form-drawer' } }}><Box className="drawer-header"><Box><Typography className="eyebrow">{eyebrow}</Typography><Typography variant="h2">{title}</Typography></Box><IconButton onClick={onClose} aria-label="关闭"><CloseRounded /></IconButton></Box><Box className="drawer-body">{children}</Box></Drawer>
}
