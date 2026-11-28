// What can.ts loads on demand: the session's `can`. A module of its own, which
// nothing imports statically, so that an app that also imports session.ts
// (useSession, Guard) has no module both imported and loaded on demand,
// which a build warns about; session.ts comes with it only when nothing else
// has brought it.
import { useSession } from './session.ts'

/** The signed-in user's `can`, asking for the session if nothing has yet. */
export const sessionCan = (): ((permission: string) => boolean) => useSession().can
