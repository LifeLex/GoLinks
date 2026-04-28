import { Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { FileText, Trash2 } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { DocUploader } from "@/components/DocUploader";
import { api, ApiError } from "@/lib/api";

export function DocsListPage() {
  const qc = useQueryClient();
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

  return (
    <div className="mx-auto max-w-3xl space-y-6 px-4 py-8">
      <header className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-light tracking-tight">Docs</h1>
          <p className="text-sm text-muted-foreground">
            Markdown and MDX documents rendered with shadcn primitives.
          </p>
        </div>
        <DocUploader />
      </header>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Couldn't load documents</AlertTitle>
          <AlertDescription>
            {error instanceof Error ? error.message : String(error)}
          </AlertDescription>
        </Alert>
      )}

      {isLoading ? (
        <div className="space-y-2">
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
        </div>
      ) : (data?.documents?.length ?? 0) === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center gap-3 py-8 text-center text-sm text-muted-foreground">
            <FileText className="h-8 w-8" />
            No documents yet. Upload a <code className="font-mono">.md</code> or{" "}
            <code className="font-mono">.mdx</code> file to get started.
          </CardContent>
        </Card>
      ) : (
        <ul className="space-y-2">
          {data!.documents.map((doc) => (
            <li key={doc.path}>
              <Card className="transition-colors hover:border-accent">
                <CardContent className="flex items-center justify-between gap-4 py-4">
                  <Link
                    to={`/docs/${encodeURIComponent(doc.path)}`}
                    className="flex flex-1 items-center gap-3"
                  >
                    <FileText className="h-5 w-5 text-muted-foreground" />
                    <div className="flex flex-col">
                      <span className="font-medium text-foreground">{doc.title}</span>
                      <span className="font-mono text-xs text-muted-foreground">{doc.path}</span>
                    </div>
                    <span className="ml-2 rounded-sm bg-secondary px-1.5 py-0.5 text-xs uppercase text-muted-foreground">
                      {doc.type}
                    </span>
                  </Link>
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label="Delete document"
                    onClick={() => {
                      if (confirm(`Delete ${doc.path}?`)) {
                        deleteMutation.mutate(doc.path);
                      }
                    }}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </CardContent>
              </Card>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
