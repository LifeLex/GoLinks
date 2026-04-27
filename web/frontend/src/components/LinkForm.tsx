import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

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
});

type FormValues = z.infer<typeof schema>;

export function LinkForm() {
  const qc = useQueryClient();
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { word: "", link: "" },
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

  return (
    <form
      onSubmit={handleSubmit((values) => mutation.mutate(values))}
      className="space-y-3"
      noValidate
    >
      <div className="flex flex-col gap-0 overflow-hidden rounded-lg border border-border bg-card shadow-sm sm:flex-row">
        <div className="flex-1">
          <Label htmlFor="word" className="sr-only">
            Keyword
          </Label>
          <Input
            id="word"
            placeholder="Keyword"
            autoComplete="off"
            className="h-12 rounded-none border-0 border-b border-border bg-transparent text-base shadow-none focus-visible:ring-0 sm:border-b-0 sm:border-r"
            {...register("word")}
          />
        </div>
        <div className="flex-[2]">
          <Label htmlFor="link" className="sr-only">
            URL
          </Label>
          <Input
            id="link"
            placeholder="URL (use {*} for a variable)"
            autoComplete="off"
            className="h-12 rounded-none border-0 bg-transparent text-base shadow-none focus-visible:ring-0"
            {...register("link")}
          />
        </div>
        <Button
          type="submit"
          size="lg"
          className="h-12 rounded-none px-6 sm:w-auto"
          disabled={isSubmitting || mutation.isPending}
        >
          {mutation.isPending ? "Adding…" : "Add Link"}
        </Button>
      </div>
      {(errors.word || errors.link) && (
        <p className="text-sm text-destructive">
          {errors.word?.message ?? errors.link?.message}
        </p>
      )}
    </form>
  );
}
