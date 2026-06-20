import { memo } from 'react'

interface FooterProps {
  totalArticles: number
}

function Footer({ totalArticles }: FooterProps) {
  return (
    <div className="flex flex-col items-center gap-1 py-8">
      <p className="font-mono text-caption uppercase tracking-widest text-slate-500/60">
        {totalArticles} articles indexed
      </p>
    </div>
  )
}

export default memo(Footer)
