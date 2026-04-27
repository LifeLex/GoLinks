import * as React from "react";

import { compileMDX } from "@/lib/mdx";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert";

type Props = { source: string };

export function MDXRenderer({ source }: Props) {
  const [Content, setContent] = React.useState<React.ComponentType | null>(null);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    let cancelled = false;
    setContent(null);
    setError(null);

    compileMDX(source)
      .then((mod) => {
        if (cancelled) return;
        setContent(() => mod.default);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : String(err));
      });

    return () => {
      cancelled = true;
    };
  }, [source]);

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Failed to render document</AlertTitle>
        <AlertDescription>
          <pre className="whitespace-pre-wrap text-xs">{error}</pre>
        </AlertDescription>
      </Alert>
    );
  }

  if (!Content) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-8 w-2/3" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-5/6" />
        <Skeleton className="h-4 w-4/6" />
      </div>
    );
  }

  return (
    <article className="prose prose-neutral max-w-none">
      <Content />
    </article>
  );
}
