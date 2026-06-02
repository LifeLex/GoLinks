import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ArrowRight } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api, ApiError } from "@/lib/api";

const schema = z.object({
  word: z
    .string()
    .trim()
    .min(1, "Please enter a keyword.")
    .refine((v) => !v.endsWith("/"), "Keywords ending in '/' are not supported."),
  link: z.string().trim().min(1, "Please enter a URL."),
  tags: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

/** Split a comma-separated tag string into a clean array. */
function parseTags(raw?: string): string[] {
  if (!raw) return [];
  return raw
    .split(",")
    .map((t) => t.trim())
    .filter(Boolean);
}

export function LinkForm() {
  const qc = useQueryClient();
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { word: "", link: "", tags: "" },
  });

  const mutation = useMutation({
    mutationFn: api.createLink,
    onSuccess: (_, vars) => {
      toast.success(`Added keyword "${vars.word}"`);
      reset();
      qc.invalidateQueries({ queryKey: ["links"] });
    },
    onError: (err: unknown) => {
      const msg = err instanceof ApiError ? err.message : "Failed to add link";
      toast.error(msg);
    },
  });

  const inputClass =
    "h-7 rounded-none border-0 bg-transparent px-0 text-base shadow-none focus-visible:ring-0 focus-visible:ring-offset-0";

  return (
    <form
      onSubmit={handleSubmit((values) =>
        mutation.mutate({ word: values.word, link: values.link, tags: parseTags(values.tags) }),
      )}
      className="space-y-3"
      noValidate
    >
      <div className="panel flex flex-col overflow-hidden rounded-md focus-within:border-foreground/30 sm:flex-row">
        <div className="flex-1 border-b border-border px-4 py-2.5 sm:border-b-0 sm:border-r">
          <Label htmlFor="word" className="label-mono">
            Keyword
          </Label>
          <Input
            id="word"
            placeholder="github"
            autoComplete="off"
            className={inputClass}
            {...register("word")}
          />
        </div>
        <div className="flex-[2] border-b border-border px-4 py-2.5 sm:border-b-0 sm:border-r">
          <Label htmlFor="link" className="label-mono">
            Target&nbsp;URL
          </Label>
          <Input
            id="link"
            placeholder="https://github.com/search?q={*}"
            autoComplete="off"
            className={inputClass}
            {...register("link")}
          />
        </div>
        <Button
          type="submit"
          className="group h-auto rounded-none px-6 py-3 font-mono text-xs font-bold uppercase tracking-[0.14em] sm:py-0"
          disabled={isSubmitting || mutation.isPending}
        >
          {mutation.isPending ? "Adding…" : "Add"}
          <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
        </Button>
      </div>
      <div className="panel overflow-hidden rounded-md px-4 py-2.5 focus-within:border-foreground/30">
        <Label htmlFor="tags" className="label-mono">
          Tags&nbsp;<span className="text-muted-foreground/60">(optional, comma-separated)</span>
        </Label>
        <Input
          id="tags"
          placeholder="infra, monitoring"
          autoComplete="off"
          className={inputClass}
          {...register("tags")}
        />
      </div>
      {(errors.word || errors.link) && (
        <p className="flex items-center gap-2 font-mono text-xs text-destructive">
          <span className="h-1.5 w-1.5 bg-destructive" />
          {errors.word?.message ?? errors.link?.message}
        </p>
      )}
    </form>
  );
}
