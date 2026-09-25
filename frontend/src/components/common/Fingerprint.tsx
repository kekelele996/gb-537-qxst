import { ContentCopyRounded } from '@mui/icons-material'
import { IconButton, Tooltip } from '@mui/material'
import { useState } from 'react'

export function Fingerprint({ value, compact = false }: { value: string; compact?: boolean }) {
  const [copied, setCopied] = useState(false)
  const shown = compact && value.length > 24 ? `${value.slice(0, 12)}…${value.slice(-10)}` : value
  const copy = async () => {
    await navigator.clipboard.writeText(value)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1500)
  }
  return (
    <span className="fingerprint">
      <span title={value}>{shown}</span>
      <Tooltip title={copied ? '已复制' : '复制指纹'}>
        <IconButton size="small" onClick={copy} aria-label="复制指纹"><ContentCopyRounded fontSize="inherit" /></IconButton>
      </Tooltip>
    </span>
  )
}

