import type { PopularQuery } from "@/lib/api";

type Props = { queries: PopularQuery[] };

export function RecentQueries({ queries }: Props) {
  if (queries.length === 0) {
    return null;
  }

  const max = Math.max(...queries.map((q) => q.count), 1);

  return (
    <div className="panel overflow-hidden rounded-md">
      <ul>
        {queries.map((q, i) => (
          <li
            key={q.word}
            className="relative flex items-center gap-3 border-b border-border px-4 py-3 last:border-b-0"
          >
            {/* proportional usage bar — a Braun data readout */}
            <span
              className="pointer-events-none absolute inset-y-0 left-0 bg-primary/[0.07]"
              style={{ width: `${(q.count / max) * 100}%` }}
              aria-hidden
            />
            <span className="relative w-6 font-mono text-xs tabular-nums text-muted-foreground">
              {String(i + 1).padStart(2, "0")}
            </span>
            <span className="relative font-mono text-sm font-medium text-foreground">{q.word}</span>
            <span className="relative min-w-0 flex-1 truncate font-mono text-xs text-muted-foreground">
              <span className="text-border">→</span> {q.link}
            </span>
            <span className="relative font-mono text-xs font-bold tabular-nums text-primary">
              ×{q.count}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
