import type { KeywordInfo } from "@/lib/api";

type Props = {
  keywords: KeywordInfo[];
  /** When provided, tag chips become buttons that trigger a search for the tag. */
  onTagClick?: (tag: string) => void;
};

function TagChips({ tags, onTagClick }: { tags: string[]; onTagClick?: (tag: string) => void }) {
  if (tags.length === 0) return null;
  return (
    <span className="mt-1.5 flex flex-wrap gap-1">
      {tags.map((tag) =>
        onTagClick ? (
          <button
            key={tag}
            type="button"
            onClick={() => onTagClick(tag)}
            className="rounded-sm border border-border bg-secondary px-1.5 py-0.5 font-mono text-[0.7rem] text-muted-foreground transition-colors hover:border-primary hover:text-primary"
          >
            {tag}
          </button>
        ) : (
          <span
            key={tag}
            className="rounded-sm border border-border bg-secondary px-1.5 py-0.5 font-mono text-[0.7rem] text-muted-foreground"
          >
            {tag}
          </span>
        ),
      )}
    </span>
  );
}

function formatDate(iso: string) {
  if (!iso) return "—";
  try {
    return new Date(iso).toISOString().slice(0, 10);
  } catch {
    return iso;
  }
}

function isExternal(link: string) {
  return link.startsWith("http://") || link.startsWith("https://");
}

const GRID = "grid grid-cols-[minmax(0,1fr)_minmax(0,1.8fr)_auto] gap-4";

export function KeywordTable({ keywords, onTagClick }: Props) {
  if (keywords.length === 0) {
    return (
      <div className="panel flex flex-col items-center gap-3 rounded-md px-4 py-12 text-center">
        <span className="h-3 w-3 bg-primary" />
        <p className="font-mono text-sm text-muted-foreground">
          No keywords yet — add your first one above.
        </p>
      </div>
    );
  }

  return (
    <div className="panel overflow-hidden rounded-md">
      {/* column labels */}
      <div className={`${GRID} items-center border-b border-border bg-secondary/40 px-4 py-2.5`}>
        <span className="label-mono">Keyword</span>
        <span className="label-mono">Target</span>
        <span className="label-mono text-right">Created</span>
      </div>

      <ul>
        {keywords.map((k) => (
          <li
            key={`${k.word}-${k.created_at}`}
            className="group relative border-b border-border last:border-b-0 transition-colors hover:bg-secondary/30"
          >
            {/* hover marker — orange tick slides in from the edge */}
            <span className="absolute inset-y-0 left-0 w-0.5 origin-center scale-y-0 bg-primary transition-transform duration-150 group-hover:scale-y-100" />
            <div className={`${GRID} items-start px-4 py-3`}>
              <span className="min-w-0 font-mono text-sm">
                <span className="block truncate">
                  <span className="text-muted-foreground">go&nbsp;</span>
                  <span className="font-medium text-foreground">{k.word}</span>
                </span>
                <TagChips tags={k.tags ?? []} onTagClick={onTagClick} />
              </span>
              <span className="truncate pt-0.5 font-mono text-xs text-muted-foreground">
                {isExternal(k.link) ? (
                  <a
                    href={k.link}
                    className="underline decoration-transparent underline-offset-2 transition-colors hover:text-primary hover:decoration-primary"
                  >
                    {k.link}
                  </a>
                ) : (
                  k.link
                )}
              </span>
              <span className="tnum text-right font-mono text-xs text-muted-foreground">
                {formatDate(k.created_at)}
              </span>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
