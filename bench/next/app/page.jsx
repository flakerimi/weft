// The benchmark's landing, the same content as bench/weft/pages/index.vue, as
// a server component with the counter as its one client component.
import content from '../../content.json'
import Counter from './Counter.jsx'

export default function Page() {
  return (
    <main className="page">
      <header className="hero">
        <span className="tag">{content.hero.tag}</span>
        <h1 className="title">{content.hero.title}</h1>
        <p className="lead">{content.hero.lead}</p>
        <p className="sub">{content.hero.sub}</p>
      </header>

      {/* The WebP bench/tools/images.mjs makes, with the attributes Astro's
          <Image> writes, as on the Weft page. A static export cannot use
          next/image's optimizer, so both Next builds take the same file. */}
      <img
        className="image"
        src="/hero.webp"
        alt={content.image.alt}
        width={content.image.width}
        height={content.image.height}
        loading="lazy"
        decoding="async"
      />

      <section>
        <h2>Features</h2>
        <div className="grid">
          {content.features.map((f) => (
            <article className="card" key={f.title}>
              <h3>{f.title}</h3>
              <p className="muted">{f.text}</p>
            </article>
          ))}
        </div>
      </section>

      <section>
        <h2>Pricing</h2>
        <div className="grid">
          {content.pricing.map((p) => (
            <article className="card" key={p.name}>
              <h3>{p.name}</h3>
              <p className="price">
                {p.price} <span className="muted">{p.period}</span>
              </p>
              <ul className="list">
                {p.items.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </article>
          ))}
        </div>
      </section>

      <section>
        <h2>{content.counter.title}</h2>
        <p className="muted">{content.counter.description}</p>
        <Counter start={0} />
      </section>

      <footer className="footer muted">{content.footer}</footer>
    </main>
  )
}
