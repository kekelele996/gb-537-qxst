import { AltRouteRounded, ArrowForwardRounded } from '@mui/icons-material'
import { Box, Typography } from '@mui/material'
import type { DependentService } from '../../types/dependent-service'

const criticalityLabels = { low: '低', medium: '中', high: '高', critical: '关键' }

export function DependencyGraph({ services, highlightedIds = [] }: { services: DependentService[]; highlightedIds?: number[] }) {
  const byId = new Map(services.map((service) => [service.id, service]))
  if (!services.length) return <Box className="graph-empty"><AltRouteRounded /><span>暂无依赖路径</span></Box>
  return (
    <Box className="dependency-graph" role="list" aria-label="服务依赖路径">
      {services.map((service) => (
        <Box className={`graph-row ${highlightedIds.includes(service.id) ? 'is-affected' : ''}`} key={service.id} role="listitem">
          <Box className="graph-node">
            <Box className="graph-node-marker">{service.service_code.slice(0, 2)}</Box>
            <Box className="graph-node-copy">
              <Typography component="strong">{service.service_code}</Typography>
              <Typography component="span">{service.name}</Typography>
            </Box>
            <Box className={`criticality criticality-${service.criticality}`}>{criticalityLabels[service.criticality]}</Box>
          </Box>
          <Box className="graph-edges">
            {service.dependency_edges_json.length ? service.dependency_edges_json.map((edge) => (
              <Box className="graph-edge" key={edge}><ArrowForwardRounded fontSize="small" /><span>{byId.get(edge)?.service_code ?? `#${edge}`}</span></Box>
            )) : <span className="graph-terminal">终端服务</span>}
          </Box>
        </Box>
      ))}
    </Box>
  )
}

