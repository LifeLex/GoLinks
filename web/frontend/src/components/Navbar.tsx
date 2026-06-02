import { NavLink, useNavigate } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { cn } from "@/lib/utils";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/auth/AuthContext";

const links = [
  { to: "/", label: "Index", end: true },
  { to: "/setup", label: "Setup" },
  { to: "/docs", label: "Docs" },
];

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "relative px-3 py-2 font-mono text-xs font-medium uppercase tracking-[0.14em] transition-colors",
    isActive ? "text-background" : "text-background/55 hover:text-background",
  );

function UnderlinedLabel({ label, isActive }: { label: string; isActive: boolean }) {
  return (
    <>
      {label}
      <span
        className={cn(
          "absolute inset-x-3 -bottom-px h-0.5 bg-primary transition-transform duration-200",
          isActive ? "scale-x-100" : "scale-x-0",
        )}
      />
    </>
  );
}

/** Right-hand auth cluster: user email + admin link + logout, or a Sign in link. */
function AuthCluster() {
  const { isAuthenticated, isAdmin, user, refresh } = useAuth();
  const qc = useQueryClient();
  const navigate = useNavigate();

  const logout = useMutation({
    mutationFn: api.logout,
    onSuccess: async () => {
      await refresh();
      qc.clear();
      toast.success("Signed out");
      navigate("/");
    },
    onError: (err: unknown) => {
      toast.error(err instanceof ApiError ? err.message : "Logout failed");
    },
  });

  if (!isAuthenticated) {
    return (
      <NavLink to="/login" className={navLinkClass}>
        {({ isActive }) => <UnderlinedLabel label="Sign in" isActive={isActive} />}
      </NavLink>
    );
  }

  return (
    <div className="flex items-center gap-1 sm:gap-2">
      {isAdmin && (
        <NavLink to="/admin/users" className={navLinkClass}>
          {({ isActive }) => <UnderlinedLabel label="Users" isActive={isActive} />}
        </NavLink>
      )}
      <span className="hidden font-mono text-xs text-background/55 sm:inline">{user?.email}</span>
      <button
        type="button"
        onClick={() => logout.mutate()}
        disabled={logout.isPending}
        className="px-3 py-2 font-mono text-xs font-medium uppercase tracking-[0.14em] text-background/55 transition-colors hover:text-primary"
      >
        Logout
      </button>
    </div>
  );
}

export function Navbar() {
  return (
    <nav className="sticky top-0 z-50 border-b border-primary/80 bg-foreground text-background">
      <div className="mx-auto flex h-16 max-w-[1180px] items-center justify-between px-4 sm:px-6">
        {/* Nameplate */}
        <NavLink to="/" className="group flex items-center gap-2.5">
          <span className="h-2.5 w-2.5 bg-primary transition-transform duration-300 group-hover:rotate-90" />
          <span className="display text-lg font-bold tracking-tight">
            go<span className="text-primary">links</span>
          </span>
        </NavLink>

        {/* Control cluster */}
        <div className="flex items-center gap-1 sm:gap-2">
          {links.map((l) => (
            <NavLink key={l.to} to={l.to} end={l.end} className={navLinkClass}>
              {({ isActive }) => <UnderlinedLabel label={l.label} isActive={isActive} />}
            </NavLink>
          ))}
          <span className="mx-1 hidden h-4 w-px bg-background/20 sm:block" />
          <AuthCluster />
        </div>
      </div>
    </nav>
  );
}
