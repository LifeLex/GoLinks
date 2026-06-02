import { Link, useLocation } from "react-router-dom";

import { Button } from "@/components/ui/button";

export function NotFoundPage() {
  const { pathname } = useLocation();

  return (
    <div className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-xl items-center px-4 sm:px-6">
      <div className="w-full space-y-8">
        <div className="space-y-3">
          <p className="label-mono text-primary">Error</p>
          <div className="display text-8xl font-bold leading-none tracking-tight tnum">404</div>
          <h1 className="font-mono text-sm font-bold uppercase tracking-[0.18em] text-foreground">
            Page not found
          </h1>
        </div>

        <p className="font-mono text-sm text-muted-foreground">
          No route matches{" "}
          <span className="rounded-sm border border-border bg-secondary px-1.5 py-0.5 text-foreground">
            {pathname}
          </span>
        </p>

        <div className="flex flex-wrap gap-3">
          <Button asChild>
            <Link to="/">Go home</Link>
          </Button>
          <Button asChild variant="outline">
            <Link to="/setup">Setup guide</Link>
          </Button>
        </div>

        <div className="panel space-y-3 rounded-md p-5 text-sm">
          <p className="label-mono">If you were using a golink</p>
          <ul className="space-y-2 text-muted-foreground">
            <li className="flex gap-3">
              <span className="mt-2 h-1 w-1 shrink-0 bg-primary" />
              <span>
                Check the keyword is spelled correctly — browse the{" "}
                <Link
                  to="/"
                  className="font-medium text-foreground underline decoration-border underline-offset-4 transition-colors hover:text-primary hover:decoration-primary"
                >
                  keyword index
                </Link>
                .
              </span>
            </li>
            <li className="flex gap-3">
              <span className="mt-2 h-1 w-1 shrink-0 bg-primary" />
              <span>Add it on the home page if it doesn&rsquo;t exist yet.</span>
            </li>
          </ul>
        </div>
      </div>
    </div>
  );
}
