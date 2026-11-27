---
group: basics
purpose: A person or thing as a small round picture, falling back to initials or an icon.
use: Beside a name in a list, a comment or a menu, and in an AvatarGroup for the people on something.
avoid: For a logo or a photo that is content, use an img; for a status on the avatar, wrap it in a Chip.
adapts:
  input: Nothing changes; an avatar is not a hit target, so put it in a Button or Link for one.
  surface: Nothing changes.
  width: Nothing changes.
---
It shows `src` when it loads. With no `src`, or when the image fails (also
when it failed before an island hydrated), it shows initials: `text` if
given, else the first letters of the first and last words of `alt`
("Ada Lovelace" is AL). With neither, it shows `icon`, a Lucide name, or
without one a silhouette it draws itself (so no icon ships for it).

`alt` names it: the image's alt text, or, for initials and icon, the
avatar's `aria-label` with `role="img"`. Without `alt` it is decorative.
`size` is `sm` (24 px), `md` (32 px) or `lg` (40 px); in an AvatarGroup the
group's size wins.
