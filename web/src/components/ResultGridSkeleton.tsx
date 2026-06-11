export default function ResultGridSkeleton({ count = 6 }: { count?: number }) {
  return (
    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      {Array.from({ length: count }).map((_, index) => (
        <div
          key={index}
          className="overflow-hidden rounded-3xl border border-white/10 bg-gradient-to-br from-surface-900/80 via-surface-900/70 to-surface-950/70 shadow-[0_18px_50px_rgba(2,6,23,0.35)]"
        >
          <div className="p-5 sm:p-6">
            <div className="flex items-start justify-between gap-3">
              <div className="flex flex-wrap items-center gap-2">
                <div className="h-6 w-24 rounded-full skeleton-shimmer" />
                <div className="h-6 w-16 rounded-full skeleton-shimmer" />
              </div>
              <div className="h-6 w-16 rounded-full skeleton-shimmer" />
            </div>
            <div className="mt-5 h-5 w-5/6 rounded-full skeleton-shimmer" />
            <div className="mt-3 h-4 w-full rounded-full skeleton-shimmer" />
            <div className="mt-2 h-4 w-11/12 rounded-full skeleton-shimmer" />
            <div className="mt-4 flex flex-wrap gap-2">
              <div className="h-7 w-24 rounded-full skeleton-shimmer" />
              <div className="h-7 w-28 rounded-full skeleton-shimmer" />
              <div className="h-7 w-20 rounded-full skeleton-shimmer" />
            </div>
            <div className="mt-5 h-9 w-32 rounded-full skeleton-shimmer" />
            <div className="mt-4 h-24 rounded-2xl skeleton-shimmer" />
          </div>
        </div>
      ))}
    </div>
  );
}
