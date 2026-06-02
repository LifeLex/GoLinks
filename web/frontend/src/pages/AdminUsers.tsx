import * as React from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Trash2 } from "lucide-react";

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
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { api, ApiError, type User } from "@/lib/api";
import { useAuth } from "@/auth/AuthContext";

const schema = z.object({
  email: z.string().trim().email("Enter a valid email."),
  password: z.string().min(8, "Use at least 8 characters."),
  role: z.enum(["user", "admin"]),
});

type FormValues = z.infer<typeof schema>;

function AddUserDialog() {
  const qc = useQueryClient();
  const [open, setOpen] = React.useState(false);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: "", password: "", role: "user" },
  });

  const mutation = useMutation({
    mutationFn: api.createUser,
    onSuccess: (_, vars) => {
      toast.success(`Added ${vars.email}`);
      reset();
      setOpen(false);
      qc.invalidateQueries({ queryKey: ["users"] });
    },
    onError: (err: unknown) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to add user");
    },
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>Add user</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add user</DialogTitle>
          <DialogDescription>Create a new account and assign a role.</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit((v) => mutation.mutate(v))} className="space-y-4" noValidate>
          <div className="space-y-1.5">
            <Label htmlFor="new-email" className="label-mono">
              Email
            </Label>
            <Input id="new-email" type="email" autoComplete="off" {...register("email")} />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="new-password" className="label-mono">
              Password
            </Label>
            <Input
              id="new-password"
              type="password"
              autoComplete="new-password"
              {...register("password")}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="new-role" className="label-mono">
              Role
            </Label>
            <select
              id="new-role"
              className="h-9 w-full rounded-md border border-border bg-transparent px-3 font-mono text-sm"
              {...register("role")}
            >
              <option value="user">user</option>
              <option value="admin">admin</option>
            </select>
          </div>
          {(errors.email || errors.password) && (
            <p className="flex items-center gap-2 font-mono text-xs text-destructive">
              <span className="h-1.5 w-1.5 bg-destructive" />
              {errors.email?.message ?? errors.password?.message}
            </p>
          )}
          <DialogFooter>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "Adding…" : "Add user"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export function AdminUsersPage() {
  const qc = useQueryClient();
  const { user: current } = useAuth();

  const { data, isLoading } = useQuery({
    queryKey: ["users"],
    queryFn: api.listUsers,
  });

  const remove = useMutation({
    mutationFn: api.deleteUser,
    onSuccess: () => {
      toast.success("User removed");
      qc.invalidateQueries({ queryKey: ["users"] });
    },
    onError: (err: unknown) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to remove user");
    },
  });

  const users: User[] = data?.users ?? [];

  return (
    <div className="mx-auto max-w-3xl px-4 py-12 sm:px-6">
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <p className="label-mono">Administration</p>
            <h1 className="display text-3xl font-bold tracking-tight">Users</h1>
          </div>
          <AddUserDialog />
        </div>

        {isLoading ? (
          <div className="space-y-px">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : (
          <div className="panel overflow-hidden rounded-md">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Email</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {users.map((u) => (
                  <TableRow key={u.id}>
                    <TableCell className="font-mono text-sm">{u.email}</TableCell>
                    <TableCell className="font-mono text-xs uppercase text-muted-foreground">
                      {u.role}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`Remove ${u.email}`}
                        disabled={u.id === current?.id || remove.isPending}
                        onClick={() => remove.mutate(u.id)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    </div>
  );
}
