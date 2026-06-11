import type { Result } from "@/lib/types";
import ResultCard from "./ResultCard";

interface Props {
  results: Result[];
}

export default function ResultGrid({ results }: Props) {
  if (results.length === 0) return null;

  return (
    <div className="grid items-stretch gap-4 md:grid-cols-2 xl:grid-cols-3">
      {results.map((r, index) => (
        <div
          key={r.id}
          className="result-enter h-full"
          style={{ animationDelay: `${index * 55}ms` }}
        >
          <ResultCard result={r} />
        </div>
      ))}
    </div>
  );
}
