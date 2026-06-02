import type { ReactNode } from "react";

type Props = {
  /** Two-digit section index, e.g. "01". */
  index: string;
  title: string;
  /** Optional right-aligned meta (counts, controls). */
  meta?: ReactNode;
};

/**
 * SectionHeader renders an engineering-manual section label: an orange
 * index, a mono caps title, and a hairline rule running to the edge.
 * It is the structural motif that divides the instrument grid.
 */
export function SectionHeader({ index, title, meta }: Props) {
  return (
    <div className="flex items-center gap-3">
      <span className="font-mono text-xs font-bold tabular-nums text-primary">{index}</span>
      <h2 className="font-mono text-xs font-bold uppercase tracking-[0.18em] text-foreground">
        {title}
      </h2>
      <span className="h-px flex-1 origin-left bg-border" />
      {meta && <span className="label-mono shrink-0">{meta}</span>}
    </div>
  );
}
