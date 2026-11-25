'use client'
// The benchmark's island, as a client component: the only part of the page
// React hydrates with state. Its styles are in bench/shared/landing.css.
import { useState } from 'react'

export default function Counter({ start }) {
  const [count, setCount] = useState(start)
  return (
    <div className="counter">
      <button type="button" className="counter__button" aria-label="Decrease" onClick={() => setCount(count - 1)}>
        −
      </button>
      <output className="counter__value" aria-live="polite">
        {count}
      </output>
      <button type="button" className="counter__button" aria-label="Increase" onClick={() => setCount(count + 1)}>
        +
      </button>
    </div>
  )
}
