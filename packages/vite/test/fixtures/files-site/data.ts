// Highlighted once, when the page build loads this module.
import { highlightCode } from 'virtual:weft/content'

export const code = await highlightCode('const a = 1', 'ts')
