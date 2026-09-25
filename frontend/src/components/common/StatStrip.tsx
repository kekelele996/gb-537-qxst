import type { ReactNode } from 'react'
import { Box, Typography } from '@mui/material'

export function StatStrip({ items }: { items: { label: string; value: ReactNode; tone?: string }[] }) {
  return <Box className="stat-strip">{items.map((item) => <Box className={`stat-item ${item.tone ?? ''}`} key={item.label}><Typography>{item.label}</Typography><Box className="stat-value">{item.value}</Box></Box>)}</Box>
}

