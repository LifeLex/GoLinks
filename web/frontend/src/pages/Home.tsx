import { useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { Check, Copy } from "lucide-react";
import * as React from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { KeywordTable } from "@/components/KeywordTable";
import { LinkForm } from "@/components/LinkForm";
import { RecentQueries } from "@/components/RecentQueries";
import { Omnibox } from "@/components/Omnibox";
import { SearchBox } from "@/components/SearchBox";
import { SectionHeader } from "@/components/SectionHeader";
import { api } from "@/lib/api";
import { useAuth } from "@/auth/AuthContext";

function Mono({ children }: { children: React.ReactNode }) {
  return (
    <code className="rounded-sm border border-border bg-secondary px-1.5 py-0.5 font-mono text-[0.85em]">
      {children}
    </code>
  );
}

/** Readout strip for the search-engine URL, click-to-copy. */
function SearchEngineReadout({ baseUrl }: { baseUrl: string }) {
  const [copied, setCopied] = React.useState(false);
  const value = `${baseUrl}/query/%s`;

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      toast.success("Copied search-engine URL");
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      toast.error("Couldn't copy to clipboard");
    }
  };

  return (
    <button
      type="button"
      onClick={copy}
      className="panel group flex w-full items-center gap-3 rounded-md px-4 py-3 text-left transition-colors hover:border-foreground/30"
    >
      <span className="label-mono shrink-0">Engine URL</span>
      <span className="min-w-0 flex-1 truncate font-mono text-sm text-foreground">{value}</span>
      <span className="flex shrink-0 items-center gap-1.5 text-muted-foreground transition-colors group-hover:text-foreground">
        {copied ? <Check className="h-4 w-4 text-primary" /> : <Copy className="h-4 w-4" />}
      </span>
    </button>
  );
}

export function HomePage() {
  const [params, setParams] = useSearchParams();
  const { isAuthenticated } = useAuth();
  const missing = params.get("missing");
  const query = params.get("q") ?? "";
  const searching = query.trim().length > 0;

  const { data, isLoading, error } = useQuery({
    queryKey: ["links"],
    queryFn: api.listLinks,
  });

  const { data: searchData, isLoading: searchLoading } = useQuery({
    queryKey: ["search", query],
    queryFn: () => api.search(query),
    enabled: searching,
  });

  // Write `?q=` into the URL so searches are deep-linkable; clear it when empty.
  const setQuery = React.useCallback(
    (value: string) => {
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (value.trim()) next.set("q", value);
          else next.delete("q");
          return next;
        },
        { replace: true },
      );
    },
    [setParams],
  );

  // One-shot toast when the golink resolver bounces the user back here.
  useEffect(() => {
    if (missing) {
      toast.error(`No shortcut found for "${missing}"`);
      params.delete("missing");
      setParams(params, { replace: true });
    }
  }, [missing, params, setParams]);

  const displayedKeywords = searching ? (searchData?.results ?? []) : (data?.keywords ?? []);
  const indexLoading = searching ? searchLoading : isLoading;
  const keywordCount = displayedKeywords.length;

  return (
    <div className="mx-auto max-w-3xl px-4 py-12 sm:px-6 sm:py-16">
      <div className="space-y-14">
        <header className="animate-rise space-y-5">
          <p className="label-mono">Internal URL shortener</p>
          <h1 className="display text-6xl font-bold leading-[0.92] tracking-tight sm:text-7xl">
            go<span className="text-primary">links</span>
          </h1>
          <p className="max-w-xl text-base leading-relaxed text-muted-foreground">
            Memorable shortcuts for long URLs. Type <Mono>go &lt;keyword&gt;</Mono> in your
            browser&rsquo;s address bar — once you&rsquo;ve finished the{" "}
            <a
              href="/setup"
              className="font-medium text-foreground underline decoration-border underline-offset-4 transition-colors hover:text-primary hover:decoration-primary"
            >
              setup
            </a>
            .
          </p>
        </header>

        <div className="animate-rise" style={{ animationDelay: "70ms" }}>
          <Omnibox keywords={data?.keywords?.map((k) => k.word)} />
        </div>

        {data?.base_url && (
          <div className="animate-rise" style={{ animationDelay: "130ms" }}>
            <SearchEngineReadout baseUrl={data.base_url} />
          </div>
        )}

        {error && (
          <Alert variant="destructive">
            <AlertTitle>Couldn&rsquo;t load keywords</AlertTitle>
            <AlertDescription>
              {error instanceof Error ? error.message : String(error)}
            </AlertDescription>
          </Alert>
        )}

        <section className="animate-rise space-y-4" style={{ animationDelay: "190ms" }}>
          <SectionHeader index="01" title="New shortcut" />
          {isAuthenticated ? (
            <>
              <LinkForm />
              <p className="text-sm leading-relaxed text-muted-foreground">
                Use <Mono>{"{*}"}</Mono> in the URL for a variable — e.g.{" "}
                <Mono>https://github.com/search?q={"{*}"}</Mono>, then run{" "}
                <Mono>go github claude</Mono>.
              </p>
            </>
          ) : (
            <div className="panel rounded-md px-4 py-6 text-sm leading-relaxed text-muted-foreground">
              <a
                href="/login"
                className="font-medium text-foreground underline decoration-border underline-offset-4 transition-colors hover:text-primary hover:decoration-primary"
              >
                Sign in
              </a>{" "}
              to add shortcuts.
            </div>
          )}
        </section>

        <section className="animate-rise space-y-4" style={{ animationDelay: "250ms" }}>
          <SectionHeader
            index="02"
            title="Index"
            meta={
              searching
                ? `${keywordCount} match${keywordCount === 1 ? "" : "es"}`
                : keywordCount > 0
                  ? `${keywordCount} keyword${keywordCount === 1 ? "" : "s"}`
                  : undefined
            }
          />
          <SearchBox value={query} onChange={setQuery} />
          {indexLoading ? (
            <div className="space-y-px overflow-hidden rounded-md border border-border">
              <Skeleton className="h-12 w-full rounded-none" />
              <Skeleton className="h-12 w-full rounded-none" />
              <Skeleton className="h-12 w-full rounded-none" />
            </div>
          ) : searching && keywordCount === 0 ? (
            <div className="panel flex flex-col items-center gap-3 rounded-md px-4 py-12 text-center">
              <span className="h-3 w-3 bg-primary" />
              <p className="font-mono text-sm text-muted-foreground">
                No matches for &ldquo;{query}&rdquo;.
              </p>
            </div>
          ) : (
            <KeywordTable keywords={displayedKeywords} onTagClick={setQuery} editable={isAuthenticated} />
          )}
        </section>

        {data?.recent_queries && data.recent_queries.length > 0 && (
          <section className="animate-rise space-y-4" style={{ animationDelay: "310ms" }}>
            <SectionHeader index="03" title="Recent" />
            <RecentQueries queries={data.recent_queries} />
          </section>
        )}
      </div>
    </div>
  );
}
