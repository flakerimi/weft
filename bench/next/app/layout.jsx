import '../../shared/landing.css'
import content from '../../content.json'

export const metadata = {
  title: content.title,
  description: content.description,
  openGraph: { title: content.title, description: content.description },
}

export default function RootLayout({ children }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}
