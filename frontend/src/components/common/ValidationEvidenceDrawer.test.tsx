import { render, screen } from '@testing-library/react'
import { ValidationEvidenceDrawer } from './ValidationEvidenceDrawer'

describe('ValidationEvidenceDrawer', () => {
  it('shows chain verdict and full trust path', () => {
    render(<ValidationEvidenceDrawer open onClose={() => undefined} title="PLATFORM-TLS" chainEvidence={{ valid: true, verified_at: '2030-01-01T00:00:00Z', leaf_subject: 'CN=service', root_subject: 'CN=root', path_subjects: ['CN=service', 'CN=root'], message: 'certificate chain verified' }} />)
    expect(screen.getByText('离线链验证通过')).toBeInTheDocument()
    expect(screen.getByText('certificate chain verified')).toBeInTheDocument()
    expect(screen.getAllByText('CN=service').length).toBeGreaterThan(0)
    expect(screen.getAllByText('CN=root').length).toBeGreaterThan(0)
  })
})

