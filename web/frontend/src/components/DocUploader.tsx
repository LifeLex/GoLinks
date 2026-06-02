import * as React from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Upload } from "lucide-react";

import { Button } from "@/components/ui/button";
import { api, ApiError } from "@/lib/api";

export function DocUploader() {
  const inputRef = React.useRef<HTMLInputElement>(null);
  const qc = useQueryClient();

  const mutation = useMutation({
    mutationFn: api.uploadDoc,
    onSuccess: (data) => {
      toast.success(`Uploaded ${data.filename}`);
      qc.invalidateQueries({ queryKey: ["docs"] });
    },
    onError: (err: unknown) => {
      const msg = err instanceof ApiError ? err.message : "Upload failed";
      toast.error(msg);
    },
  });

  return (
    <>
      <input
        ref={inputRef}
        type="file"
        accept=".md,.mdx"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) mutation.mutate(file);
          e.target.value = "";
        }}
      />
      <Button
        variant="outline"
        onClick={() => inputRef.current?.click()}
        disabled={mutation.isPending}
        className="shrink-0 font-mono text-xs uppercase tracking-[0.14em]"
      >
        <Upload className="h-4 w-4" />
        {mutation.isPending ? "Uploading…" : "Upload"}
      </Button>
    </>
  );
}
