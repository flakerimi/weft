---
group: navigation
purpose: A hierarchy of nodes that open and close, such as files in folders.
use: For a file browser, a table of contents, categories within categories.
avoid: For a flat list to choose from, use Listbox; for a site's links, NavigationMenu.
adapts:
  input: Each row is at least --weft-hit-min tall on touch (44 px rows when windowed).
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: It fills its container; labels are cut with an ellipsis.
---
`items` are `{ label, value, icon, children, defaultExpanded, disabled }`.
A node's key is its `value`, else its labels from the top joined with `/`;
siblings with the same label and no value get `#2`, `#3`, … after it.
Bind the chosen node's key with `v-model` (`null` when none is; an array
with `multiple`), and the open nodes' keys with `v-model:expanded`.
`selectable: false` leaves only opening and closing. It emits `select` and
`toggle(item, open)`. The `item` slot replaces a node's label.

It follows the WAI-ARIA APG Tree View pattern, drawn as one flat list of
`treeitem`s that each state `aria-level`, `aria-setsize` and
`aria-posinset`, and `aria-expanded` when they have children; `aria-selected`
on every node while nodes can be chosen, and `aria-multiselectable` with
`multiple`. Name it with `aria-label`.

**Keyboard.** The tree is one tab stop. ArrowDown and ArrowUp move over the
visible nodes, Home and End to the first and last. ArrowRight opens a closed
node, or moves into an open one; ArrowLeft closes an open node, or moves to
its parent (both mirrored right to left). `*` opens every sibling of the
focused node. Typing jumps to a node by its label. Enter (or a press) chooses
the node and opens or closes it; Space chooses it. With `multiple`, Space
toggles a node, Shift with ArrowDown or ArrowUp moves and toggles, and
Ctrl+A chooses every visible node. Disabled nodes are reached but never
chosen, opened or closed.

**Long trees.** Past 200 visible nodes only the rows in view are rendered
(Weft's `useVirtualList`), each row a fixed height, and the tree scrolls
inside itself (`--weft-tree-height`, 20rem); the focused node is always
rendered.
