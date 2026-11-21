---
group: forms
purpose: Files to send with a form, dropped, pasted or chosen, listed with previews, sizes and, with an upload function, progress.
use: For attachments, a photo or a document in a form - a profile picture, receipts, a CV - bound as File objects or sent as each is added.
avoid: For a link to a file somewhere else, use Input with type url; for picking among files already uploaded, Select or Listbox.
adapts:
  input: On pointer it is a dashed drop zone with its text, a hint and a Browse button; files dropped on it, pasted while focus is inside, or chosen in the file dialog (the button, or a press on the zone) are added. On touch there is no zone: one button opens the platform's chooser, which offers the camera, the photo library or files as accept and capture allow (capture goes straight to a camera); the button is 44 px tall. The same list is below in both forms, and the input can change with files listed; they stay, previews and uploads included.
  surface: Nothing changes in behaviour; text follows the surface's type scale.
  width: It fills the width of its container; long file names are cut with an ellipsis.
---
Bind the files with `v-model`: an array with `multiple` (new files are added
to it), else one `File` or null (a new file replaces it). With a `name` (or
its FormField's) a native form submit sends them from the hidden file input,
which holds every file however it came. `accept` (MIME types, `image/*`,
extensions), `max-size` (bytes) and `max-files` refuse files, which `reject`
reports with a reason each (`type`, `size`, `count`): the dialog only
suggests `accept`, and drops and pastes ignore it, so every file is checked.

`layout="grid"` lists the files as tiles (a square preview, the name and
size under it, the remove button over its corner) in place of rows;
`position="inside"` puts them inside the zone, where they take the place of
its icon and text once there are some (Browse stays; a press on a file does
not open the dialog). `reset` makes a choice in the file dialog replace the
files instead of adding to them; drops and pastes still add. After every
choice in the dialog the hidden input holds exactly the files listed, so a
refused file never submits. A file dragged anywhere over the component is
kept from the browser, which would otherwise open it in place of the page,
even while it is disabled.

Each file is listed with its name and its size in the locale (`1.5 kB`, `2,5
MB`), an image with a preview: an object URL, revoked when the file is
removed, replaced or the component goes. No file is ever read. Its remove
button is a CloseButton named "Remove photo.png"; after a removal focus moves
to the next file's remove button, else the last one's, else Browse.

**Uploading.** With `upload: (file, { signal, onProgress }) => Promise`, each
file added is sent at once: its row shows a progress bar ("Uploading
photo.png", determinate once `onProgress(fraction)` reports 0 to 1), then
`data-status="done"` and an `uploaded` event with what the promise resolved
to, or "Upload failed", an `error` event and a Retry button. While it
uploads the remove button is "Cancel upload of photo.png": it aborts `signal`
and removes the file; a late answer changes nothing. Without `upload` it only
collects files for v-model and the form.

Inside a FormField the label is `for` the Browse button (a press on the label
opens the dialog), which it names with its text ("Attachments Browse"); the
hint, the field's description and error describe it. What happens (added,
removed, uploaded, failed, refused) is said in a polite live region.
`disabled` ignores drops, pastes and presses. `labels` translates every
word; slots: `leading` (the zone's icon), `label`, `description`, and `file`
(a row, given the file, its status, progress and error, `remove` and
`retry`). `open()` is exposed.
