import { Box } from '@mui/material'

const labels: Record<string, string> = {
  imported: '已导入', validated: '验证通过', deprecated: '已废止', revoked: '已撤销',
  active: '运行中', inactive: '已停用', critical: '关键', high: '高', medium: '中', low: '低',
  production: '生产', staging: '预发', development: '开发', disaster_recovery: '灾备',
}

export function ToneBadge({ value }: { value: string }) {
  return <Box component="span" className={`tone-badge tone-${value}`}>{labels[value] ?? value}</Box>
}

