import { NavLink } from "react-router-dom";

import { cn } from "@/lib/utils";

const links = [
  { to: "/", label: "Home", end: true },
  { to: "/setup", label: "Setup" },
  { to: "/docs", label: "Docs" },
];

export function Navbar() {
  return (
    <nav className="sticky top-0 z-50 bg-foreground text-background shadow-md">
      <div className="mx-auto flex min-h-[70px] max-w-[1200px] items-center justify-between px-4">
        <div className="text-2xl font-light tracking-tight">
          go<span className="font-normal text-primary">links</span>
        </div>
        <div className="flex items-center gap-6">
          {links.map((l) => (
            <NavLink
              key={l.to}
              to={l.to}
              end={l.end}
              className={({ isActive }) =>
                cn(
                  "rounded-sm px-3 py-2 text-sm font-medium transition-colors hover:bg-white/10 hover:text-background",
                  isActive
                    ? "bg-primary/10 text-primary"
                    : "text-background/70",
                )
              }
            >
              {l.label}
            </NavLink>
          ))}
        </div>
      </div>
    </nav>
  );
}
