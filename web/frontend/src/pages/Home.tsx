import { useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { KeywordTable } from "@/components/KeywordTable";
import { LinkForm } from "@/components/LinkForm";
import { RecentQueries } from "@/components/RecentQueries";
import { api } from "@/lib/api";

export function HomePage() {
  const [params, setParams] = useSearchParams();
  const missing = params.get("missing");

  const { data, isLoading, error } = useQuery({
    queryKey: ["links"],
    queryFn: api.listLinks,
  });

  // One-shot toast when the golink resolver bounces the user back here.
  useEffect(() => {
    if (missing) {
      toast.error(`No shortcut found for "${missing}"`);
      params.delete("missing");
      setParams(params, { replace: true });
    }
  }, [missing, params, setParams]);

  return (
    <div className="mx-auto max-w-3xl space-y-8 px-4 py-8">
      <header className="space-y-2">
        <h1 className="text-4xl font-light tracking-tight">
          go<span className="text-primary">links</span>
        </h1>
        <p className="text-muted-foreground">
          Memorable shortcuts for long URLs. Type <code className="rounded-sm bg-secondary px-1 font-mono text-sm">go &lt;keyword&gt;</code> in your browser address bar once
          you've finished the <a href="/setup" className="text-accent underline-offset-4 hover:text-primary hover:underline">setup</a>.
        </p>
        {data?.base_url && (
          <p className="text-sm text-muted-foreground">
            Your search engine should read:{" "}
            <code className="rounded-sm bg-secondary px-1 py-0.5 font-mono text-xs">
              {data.base_url}/query/%s
            </code>
          </p>
        )}
      </header>

      <section className="space-y-3">
        <h2 className="flex items-center gap-2 text-xl font-medium">
          <span className="inline-block h-5 w-1 rounded-sm bg-primary" />
          Add a new keyword
        </h2>
        <LinkForm />
        <p className="text-sm text-muted-foreground">
          Use <code className="rounded-sm bg-secondary px-1 font-mono">{"{*}"}</code> in the URL
          for a variable. Example: <code className="rounded-sm bg-secondary px-1 font-mono">https://github.com/search?q={"{*}"}</code>, then <code className="rounded-sm bg-secondary px-1 font-mono">go github claude</code>.
        </p>
      </section>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Couldn't load keywords</AlertTitle>
          <AlertDescription>{error instanceof Error ? error.message : String(error)}</AlertDescription>
        </Alert>
      )}

      <section className="space-y-3">
        <h2 className="flex items-center gap-2 text-xl font-medium">
          <span className="inline-block h-5 w-1 rounded-sm bg-primary" />
          All keywords
        </h2>
        {isLoading ? (
          <div className="space-y-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : (
          <KeywordTable keywords={data?.keywords ?? []} />
        )}
      </section>

      {data?.recent_queries && data.recent_queries.length > 0 && (
        <section>
          <RecentQueries queries={data.recent_queries} />
        </section>
      )}
    </div>
  );
}
