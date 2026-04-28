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
    <div className="mx-auto max-w-3xl space-y-6 px-4 py-8">
      <Button variant="ghost" size="sm" asChild>
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
          <Skeleton className="h-8 w-2/3" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-5/6" />
        </div>
      )}

      {data && (
        <>
          <header className="space-y-1 border-b border-border pb-4">
            <div className="text-xs uppercase tracking-wider text-muted-foreground">
              {data.metadata.type}
            </div>
            <h1 className="text-3xl font-light tracking-tight">{data.metadata.title}</h1>
            {data.metadata.description && (
              <p className="text-muted-foreground">{data.metadata.description}</p>
            )}
          </header>
          <MDXRenderer source={data.source} />
        </>
      )}
    </div>
  );
}
