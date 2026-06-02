import { Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { FileText, Trash2 } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { DocUploader } from "@/components/DocUploader";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/auth/AuthContext";

export function DocsListPage() {
  const qc = useQueryClient();
  const { isAdmin } = useAuth();
  const { data, isLoading, error } = useQuery({
    queryKey: ["docs"],
    queryFn: api.listDocs,
  });

  const deleteMutation = useMutation({
    mutationFn: api.deleteDoc,
    onSuccess: () => {
      toast.success("Document deleted");
      qc.invalidateQueries({ queryKey: ["docs"] });
    },
    onError: (err: unknown) => {
      toast.error(err instanceof ApiError ? err.message : "Delete failed");
    },
  });

  const docs = data?.documents ?? [];

  return (
    <div className="mx-auto max-w-3xl px-4 py-12 sm:px-6 sm:py-16">
      <div className="space-y-10">
        <header className="animate-rise flex items-end justify-between gap-4">
          <div className="space-y-4">
            <p className="label-mono">Library</p>
            <h1 className="display text-4xl font-bold tracking-tight sm:text-5xl">Docs</h1>
            <p className="max-w-md text-base leading-relaxed text-muted-foreground">
              Markdown and MDX documents, compiled in your browser.
            </p>
          </div>
          {isAdmin && <DocUploader />}
        </header>

        {error && (
          <Alert variant="destructive">
            <AlertTitle>Couldn&rsquo;t load documents</AlertTitle>
            <AlertDescription>
              {error instanceof Error ? error.message : String(error)}
            </AlertDescription>
          </Alert>
        )}

        {isLoading ? (
          <div className="space-y-px overflow-hidden rounded-md border border-border">
            <Skeleton className="h-16 w-full rounded-none" />
            <Skeleton className="h-16 w-full rounded-none" />
          </div>
        ) : docs.length === 0 ? (
          <div className="panel flex flex-col items-center gap-3 rounded-md px-4 py-16 text-center">
            <FileText className="h-7 w-7 text-muted-foreground" />
            <p className="font-mono text-sm text-muted-foreground">
              No documents yet. Upload a <span className="text-foreground">.md</span> or{" "}
              <span className="text-foreground">.mdx</span> file to get started.
            </p>
          </div>
        ) : (
          <ul className="animate-rise overflow-hidden rounded-md border border-border bg-card">
            {docs.map((doc) => (
              <li
                key={doc.path}
                className="group relative flex items-center gap-4 border-b border-border px-4 py-4 transition-colors last:border-b-0 hover:bg-secondary/30"
              >
                <span className="absolute inset-y-0 left-0 w-0.5 origin-center scale-y-0 bg-primary transition-transform duration-150 group-hover:scale-y-100" />
                <Link
                  to={`/docs/${encodeURIComponent(doc.path)}`}
                  className="flex min-w-0 flex-1 items-center gap-4"
                >
                  <FileText className="h-5 w-5 shrink-0 text-muted-foreground transition-colors group-hover:text-foreground" />
                  <div className="flex min-w-0 flex-col">
                    <span className="truncate font-medium text-foreground">{doc.title}</span>
                    <span className="truncate font-mono text-xs text-muted-foreground">
                      {doc.path}
                    </span>
                  </div>
                  <span className="ml-auto shrink-0 rounded-sm border border-border px-1.5 py-0.5 font-mono text-[0.625rem] font-medium uppercase tracking-[0.14em] text-muted-foreground">
                    {doc.type}
                  </span>
                </Link>
                {isAdmin && (
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label="Delete document"
                    className="shrink-0 text-muted-foreground hover:text-destructive"
                    onClick={() => {
                      if (confirm(`Delete ${doc.path}?`)) {
                        deleteMutation.mutate(doc.path);
                      }
                    }}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
