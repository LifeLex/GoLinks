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
  password: z.string().min(1, "Enter your password."),
});

type FormValues = z.infer<typeof schema>;

export function LoginPage() {
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
    mutationFn: api.login,
    onSuccess: async () => {
      await refresh();
      toast.success("Signed in");
      navigate("/");
    },
    onError: (err: unknown) => {
      const msg = err instanceof ApiError ? err.message : "Login failed";
      toast.error(msg);
    },
  });

  return (
    <div className="mx-auto max-w-md px-4 py-16 sm:px-6">
      <div className="space-y-8">
        <header className="space-y-2">
          <p className="label-mono">Sign in</p>
          <h1 className="display text-4xl font-bold tracking-tight">
            go<span className="text-primary">links</span>
          </h1>
        </header>

        <form
          onSubmit={handleSubmit((v) => mutation.mutate(v))}
          className="space-y-4"
          noValidate
        >
          <div className="space-y-1.5">
            <Label htmlFor="email" className="label-mono">
              Email
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
              autoComplete="current-password"
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
            {mutation.isPending ? "Signing in…" : "Sign in"}
          </Button>
        </form>
      </div>
    </div>
  );
}
