import type { PopularQuery } from "@/lib/api";

type Props = { queries: PopularQuery[] };

export function RecentQueries({ queries }: Props) {
  if (queries.length === 0) {
    return null;
  }

  return (
    <div className="space-y-2">
      <h3 className="text-lg font-medium">Recent queries</h3>
      <ul className="space-y-1 text-sm">
        {queries.map((q) => (
          <li key={q.word} className="flex items-baseline gap-3">
            <span className="w-8 text-right text-muted-foreground">×{q.count}</span>
            <code className="rounded-sm bg-secondary px-1.5 py-0.5 font-mono">{q.word}</code>
            <span className="text-muted-foreground">→ {q.link}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
