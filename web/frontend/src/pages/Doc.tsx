import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { MDXRenderer } from "@/components/MDXRenderer";
import { api } from "@/lib/api";

export function DocPage() {
  const { filename = "" } = useParams();
  const { data, isLoading, error } = useQuery({
    queryKey: ["doc", filename],
    queryFn: () => api.getDoc(filename),
    enabled: Boolean(filename),
  });

  return (
    <div className="mx-auto max-w-3xl px-4 py-12 sm:px-6 sm:py-16">
      <div className="space-y-8">
        <Button
          variant="ghost"
          size="sm"
          asChild
          className="-ml-2 font-mono text-xs uppercase tracking-[0.14em] text-muted-foreground hover:text-foreground"
        >
          <Link to="/docs">
            <ArrowLeft className="h-4 w-4" />
            All documents
          </Link>
        </Button>

        {error && (
          <Alert variant="destructive">
            <AlertTitle>Document not found</AlertTitle>
            <AlertDescription>
              {error instanceof Error ? error.message : String(error)}
            </AlertDescription>
          </Alert>
        )}

        {isLoading && (
          <div className="space-y-3">
            <Skeleton className="h-9 w-2/3" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-5/6" />
          </div>
        )}

        {data && (
          <>
            <header className="animate-rise space-y-3 border-b border-border pb-6">
              <div className="label-mono text-primary">{data.metadata.type}</div>
              <h1 className="display text-4xl font-bold tracking-tight">{data.metadata.title}</h1>
              {data.metadata.description && (
                <p className="text-base leading-relaxed text-muted-foreground">
                  {data.metadata.description}
                </p>
              )}
            </header>
            <div className="animate-rise" style={{ animationDelay: "70ms" }}>
              <MDXRenderer source={data.source} />
            </div>
          </>
        )}
      </div>
    </div>
  );
}
