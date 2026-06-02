import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/auth/AuthContext";

const schema = z.object({
  email: z.string().trim().email("Enter a valid email."),
  password: z.string().min(8, "Use at least 8 characters."),
});

type FormValues = z.infer<typeof schema>;

/** First-run wizard: creates the initial admin account on an empty instance. */
export function WelcomePage() {
  const { refresh } = useAuth();
  const navigate = useNavigate();
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: "", password: "" },
  });

  const mutation = useMutation({
    mutationFn: api.setup,
    onSuccess: async () => {
      await refresh();
      toast.success("Admin account created");
      navigate("/");
    },
    onError: (err: unknown) => {
      const msg = err instanceof ApiError ? err.message : "Setup failed";
      toast.error(msg);
    },
  });

  return (
    <div className="mx-auto max-w-md px-4 py-16 sm:px-6">
      <div className="space-y-8">
        <header className="space-y-3">
          <p className="label-mono">First run</p>
          <h1 className="display text-4xl font-bold tracking-tight">
            Welcome to go<span className="text-primary">links</span>
          </h1>
          <p className="text-sm leading-relaxed text-muted-foreground">
            Create the administrator account. This is a one-time step — afterwards,
            new users are added from the admin panel.
          </p>
        </header>

        <form
          onSubmit={handleSubmit((v) => mutation.mutate(v))}
          className="space-y-4"
          noValidate
        >
          <div className="space-y-1.5">
            <Label htmlFor="email" className="label-mono">
              Admin email
            </Label>
            <Input id="email" type="email" autoComplete="email" {...register("email")} />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="password" className="label-mono">
              Password
            </Label>
            <Input
              id="password"
              type="password"
              autoComplete="new-password"
              {...register("password")}
            />
          </div>
          {(errors.email || errors.password) && (
            <p className="flex items-center gap-2 font-mono text-xs text-destructive">
              <span className="h-1.5 w-1.5 bg-destructive" />
              {errors.email?.message ?? errors.password?.message}
            </p>
          )}
          <Button type="submit" className="w-full" disabled={mutation.isPending}>
            {mutation.isPending ? "Creating…" : "Create admin account"}
          </Button>
        </form>
      </div>
    </div>
  );
}
