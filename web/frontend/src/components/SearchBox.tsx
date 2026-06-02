import * as React from "react";
import { Search, X } from "lucide-react";

type Props = {
  /** The committed query, typically sourced from the URL's `?q=`. */
  value: string;
  /** Called with the new query after a short debounce, and immediately on clear. */
  onChange: (value: string) => void;
};

/**
 * SearchBox is the keyword/link/tag filter input. It keeps a local draft for
 * snappy typing and debounces commits to `onChange` so each keystroke doesn't
 * fire a request. The committed value flows back in via `value`, which lets an
 * external action (e.g. clicking a tag chip) drive the box.
 */
export function SearchBox({ value, onChange }: Props) {
  const [draft, setDraft] = React.useState(value);

  // Re-sync when the committed value changes from the outside.
  React.useEffect(() => {
    setDraft(value);
  }, [value]);

  // Debounce commits of the local draft.
  React.useEffect(() => {
    if (draft === value) return;
    const id = window.setTimeout(() => onChange(draft), 250);
    return () => window.clearTimeout(id);
  }, [draft, value, onChange]);

  return (
    <div className="panel flex items-center gap-3 rounded-md px-4 py-3 focus-within:border-foreground/30">
      <Search className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden />
      <input
        type="text"
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        placeholder="Search keywords, URLs, tags…"
        autoComplete="off"
        aria-label="Search keywords"
        className="min-w-0 flex-1 bg-transparent font-mono text-sm text-foreground placeholder:text-muted-foreground focus:outline-none"
      />
      {draft && (
        <button
          type="button"
          onClick={() => onChange("")}
          aria-label="Clear search"
          className="shrink-0 text-muted-foreground transition-colors hover:text-foreground"
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}
