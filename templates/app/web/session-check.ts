// GET /api/auth/me answers the generated Me, which @weft/ui's useSession,
// Guard and perm read as a SessionUser. This line fails pnpm typecheck when
// the two stop fitting, so a change on either side is caught here, not in a
// browser.
import type { SessionUser } from '@weft/ui'
import type { Me } from './.weft-types/api.ts'

export const meIsASessionUser = (me: Me): SessionUser => me
