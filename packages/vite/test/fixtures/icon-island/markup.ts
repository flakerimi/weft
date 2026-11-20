/** The server HTML of the Rated island, as renderPage writes it; both halves of the icon-island tests pin it. */
export const ratedIsland = (star: string) =>
  `<weft-island data-id="Rated" data-when="load"><script type="application/json">{}</script>` +
  `<button type="button"><svg class="weft-icon" xmlns="http://www.w3.org/2000/svg" viewbox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" role="img" aria-label="1 stars">${star}</svg></button></weft-island>`
