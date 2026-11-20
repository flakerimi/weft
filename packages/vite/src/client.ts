/// <reference path="./virtual.d.ts" />
// The browser entry of a built site's pages with islands: hydrate each island
// from the registry weft() serves.
import islands from 'virtual:weft/islands'
import { startIslands } from './island-runtime.ts'

startIslands(islands)
