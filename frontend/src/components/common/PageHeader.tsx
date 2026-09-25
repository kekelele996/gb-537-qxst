import type { ReactNode } from 'react'
import { Box, Typography } from '@mui/material'

export function PageHeader({ eyebrow, title, summary, actions }: { eyebrow: string; title: string; summary: string; actions?: ReactNode }) {
  return (
    <header className="page-header">
      <Box className="page-heading-copy">
        <Typography className="eyebrow">{eyebrow}</Typography>
        <Typography variant="h1">{title}</Typography>
        <Typography className="page-summary">{summary}</Typography>
      </Box>
      {actions && <Box className="page-actions">{actions}</Box>}
    </header>
  )
}

