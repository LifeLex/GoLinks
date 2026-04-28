import { Link, useLocation } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

export function NotFoundPage() {
  const { pathname } = useLocation();

  return (
    <div className="mx-auto flex min-h-[calc(100vh-70px)] max-w-xl items-center px-4">
      <div className="w-full space-y-6 text-center">
        <div className="text-7xl font-light tracking-tight text-muted-foreground">404</div>
        <h1 className="text-2xl font-medium">Page not found</h1>
        <p className="text-muted-foreground">
          Couldn't find{" "}
          <code className="rounded-sm bg-secondary px-1.5 py-0.5 font-mono text-sm">
            {pathname}
          </code>
          .
        </p>
        <div className="flex flex-wrap justify-center gap-3">
          <Button asChild>
            <Link to="/">Go home</Link>
          </Button>
          <Button asChild variant="outline">
            <Link to="/setup">Setup guide</Link>
          </Button>
        </div>
        <Card className="text-left">
          <CardContent className="space-y-2 py-4 text-sm">
            <p className="font-medium">If you were trying to use a golink:</p>
            <ul className="list-disc space-y-1 pl-5 text-muted-foreground">
              <li>
                Check the keyword is typed correctly — try the{" "}
                <Link to="/" className="text-accent underline-offset-4 hover:text-primary hover:underline">
                  keyword list
                </Link>
                .
              </li>
              <li>
                Add it on the home page if it doesn't exist yet.
              </li>
            </ul>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
