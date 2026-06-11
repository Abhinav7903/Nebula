export default function SearchStatusSkeleton() {
  return (
    <section className="overflow-hidden rounded-[2rem] border border-white/10 bg-surface-900/75 shadow-[0_24px_80px_rgba(2,6,23,0.45)] backdrop-blur-xl">
      <div className="border-b border-white/5 px-5 py-4 sm:px-6">
        <div className="inline-flex h-7 w-36 rounded-full skeleton-shimmer" />
        <div className="mt-4 h-8 w-2/3 max-w-xl rounded-2xl skeleton-shimmer" />
        <div className="mt-3 h-4 w-full max-w-2xl rounded-full skeleton-shimmer" />
        <div className="mt-2 h-4 w-5/6 max-w-xl rounded-full skeleton-shimmer" />
      </div>
      <div className="grid gap-3 px-5 py-5 sm:px-6 sm:grid-cols-3">
        <div className="h-20 rounded-2xl skeleton-shimmer" />
        <div className="h-20 rounded-2xl skeleton-shimmer" />
        <div className="h-20 rounded-2xl skeleton-shimmer" />
      </div>
      <div className="px-5 pb-5 sm:px-6">
        <div className="h-24 rounded-2xl skeleton-shimmer" />
      </div>
    </section>
  );
}
