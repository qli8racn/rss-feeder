interface FooterProps {
  totalArticles: number
}

export default function Footer({ totalArticles }: FooterProps) {
  return (
    <div className="flex flex-col items-center gap-1 py-8">
      <p className="font-mono text-[11px] uppercase tracking-widest text-slate-500/60">
        {totalArticles} articles indexed
      </p>
    </div>
  )
}
