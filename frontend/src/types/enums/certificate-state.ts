export const certificateStates = ['valid', 'expiring', 'expired', 'revoked'] as const
export type CertificateState = (typeof certificateStates)[number]

export const chainStates = ['imported', 'validated', 'deprecated', 'revoked'] as const
export type ChainState = (typeof chainStates)[number]

