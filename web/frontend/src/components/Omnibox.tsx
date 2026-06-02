import * as React from "react";
import { CornerDownLeft } from "lucide-react";

type Props = {
  /** Real keywords to cycle through; falls back to a sensible demo set. */
  keywords?: string[];
};

const FALLBACK = ["docs", "jira 4821", "github golinks", "cal", "wiki onboarding"];

/**
 * Omnibox is the hero readout: a facsimile of the browser address bar
 * typing `go <keyword>` with a blinking instrument cursor. It shows the
 * product doing its one job, literally — the thing you remember.
 */
export function Omnibox({ keywords }: Props) {
  const words = React.useMemo(() => {
    const list = (keywords ?? []).filter(Boolean).slice(0, 6);
    return list.length > 0 ? list : FALLBACK;
  }, [keywords]);

  const [text, setText] = React.useState("");
  const [wordIndex, setWordIndex] = React.useState(0);
  const [deleting, setDeleting] = React.useState(false);

  React.useEffect(() => {
    const current = words[wordIndex % words.length];
    let delay: number;

    if (!deleting && text === current) {
      delay = 1500; // hold the completed command
    } else if (deleting && text === "") {
      setDeleting(false);
      setWordIndex((i) => (i + 1) % words.length);
      delay = 350;
    } else {
      delay = deleting ? 45 : 95;
    }

    const id = window.setTimeout(() => {
      if (!deleting && text === current) {
        setDeleting(true);
      } else if (deleting) {
        setText(current.slice(0, Math.max(0, text.length - 1)));
      } else {
        setText(current.slice(0, text.length + 1));
      }
    }, delay);

    return () => window.clearTimeout(id);
  }, [text, deleting, wordIndex, words]);

  return (
    <div className="panel rounded-md shadow-[0_1px_0_0_hsl(var(--foreground)/0.04)]">
      {/* fascia label strip */}
      <div className="flex items-center justify-between border-b border-border px-4 py-2">
        <span className="label-mono">Address bar</span>
        <span className="flex items-center gap-1.5">
          <span className="h-1.5 w-1.5 rounded-full bg-border" />
          <span className="h-1.5 w-1.5 rounded-full bg-border" />
          <span className="h-1.5 w-1.5 rounded-full bg-primary" />
        </span>
      </div>

      {/* the readout */}
      <div className="flex items-center gap-3 px-4 py-5 sm:px-6 sm:py-6">
        <span className="flex h-7 items-center rounded-sm bg-primary px-2 font-mono text-xs font-bold uppercase tracking-wider text-primary-foreground">
          go
        </span>
        <div className="flex min-w-0 flex-1 items-center font-mono text-xl sm:text-2xl">
          <span className="truncate text-foreground">{text}</span>
          <span
            className="ml-0.5 inline-block h-[1.1em] w-[0.5ch] animate-blink bg-primary"
            aria-hidden
          />
        </div>
        <span className="hidden items-center gap-1.5 text-muted-foreground sm:flex">
          <span className="label-mono">Enter</span>
          <CornerDownLeft className="h-3.5 w-3.5" />
        </span>
      </div>
    </div>
  );
}
