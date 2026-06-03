import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { api, ApiError, type KeywordInfo } from "@/lib/api";

const schema = z.object({
  link: z.string().trim().min(1, "Please enter a URL."),
  tags: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

function parseTags(raw?: string): string[] {
  if (!raw) return [];
  return raw
    .split(",")
    .map((t) => t.trim())
    .filter(Boolean);
}

type Props = {
  keyword: KeywordInfo;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

/** Edit dialog for a keyword's target URL and tags. Editing appends a new
 *  revision server-side (the latest row per word wins). */
export function EditLinkDialog({ keyword, open, onOpenChange }: Props) {
  const qc = useQueryClient();
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      link: keyword.link,
      tags: (keyword.tags ?? []).join(", "),
    },
  });

  const mutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.updateLink(keyword.word, { link: values.link, tags: parseTags(values.tags) }),
    onSuccess: () => {
      toast.success(`Updated "${keyword.word}"`);
      qc.invalidateQueries({ queryKey: ["links"] });
      qc.invalidateQueries({ queryKey: ["search"] });
      onOpenChange(false);
    },
    onError: (err: unknown) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to update link");
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            Edit <span className="font-mono text-primary">go {keyword.word}</span>
          </DialogTitle>
          <DialogDescription>Change the target URL and tags. The keyword itself can't be renamed here.</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit((v) => mutation.mutate(v))} className="space-y-4" noValidate>
          <div className="space-y-1.5">
            <Label htmlFor="edit-link" className="label-mono">
              Target&nbsp;URL
            </Label>
            <Input id="edit-link" autoComplete="off" {...register("link")} />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="edit-tags" className="label-mono">
              Tags&nbsp;<span className="text-muted-foreground/60">(comma-separated)</span>
            </Label>
            <Input id="edit-tags" autoComplete="off" placeholder="infra, monitoring" {...register("tags")} />
          </div>
          {errors.link && (
            <p className="flex items-center gap-2 font-mono text-xs text-destructive">
              <span className="h-1.5 w-1.5 bg-destructive" />
              {errors.link.message}
            </p>
          )}
          <DialogFooter>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "Saving…" : "Save changes"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
