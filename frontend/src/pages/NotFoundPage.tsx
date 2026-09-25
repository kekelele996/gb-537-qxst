import { ArrowBackRounded } from '@mui/icons-material'
import { Box, Button, Typography } from '@mui/material'
import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return <Box className="not-found"><Typography className="eyebrow">ROUTE NOT FOUND</Typography><Typography variant="h1">页面不存在</Typography><Typography>该路径不在 PKI 轮换推演台的受控工作区内。</Typography><Button component={Link} to="/rollovers" variant="contained" startIcon={<ArrowBackRounded />}>返回轮换推演</Button></Box>
}

