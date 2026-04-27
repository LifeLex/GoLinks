import { useQuery } from "@tanstack/react-query";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { api } from "@/lib/api";

function BaseUrlCode() {
  const { data } = useQuery({ queryKey: ["links"], queryFn: api.listLinks });
  return (
    <code className="rounded-sm bg-secondary px-1.5 py-0.5 font-mono text-sm">
      {(data?.base_url ?? "") + "/query/%s"}
    </code>
  );
}

export function SetupPage() {
  return (
    <div className="mx-auto max-w-3xl space-y-8 px-4 py-8">
      <header className="space-y-2">
        <h1 className="text-3xl font-light tracking-tight">Browser setup</h1>
        <p className="text-muted-foreground">
          Configure your browser to use GoLinks as a search engine so typing{" "}
          <code className="rounded-sm bg-secondary px-1 font-mono">go &lt;keyword&gt;</code> in the
          address bar redirects to the matching URL.
        </p>
      </header>

      <Tabs defaultValue="chrome">
        <TabsList>
          <TabsTrigger value="chrome">Chrome / Edge</TabsTrigger>
          <TabsTrigger value="firefox">Firefox</TabsTrigger>
          <TabsTrigger value="safari">Safari</TabsTrigger>
        </TabsList>
        <TabsContent value="chrome" className="space-y-3">
          <ol className="list-decimal space-y-2 pl-6 text-sm">
            <li>Open Chrome/Edge settings.</li>
            <li>
              Go to <strong>Search engine</strong> →{" "}
              <strong>Manage search engines and site search</strong>.
            </li>
            <li>
              Click <strong>Add</strong> next to "Site search".
            </li>
            <li>
              Fill in:
              <ul className="ml-4 mt-1 list-disc space-y-1">
                <li>
                  <strong>Name:</strong> GoLinks
                </li>
                <li>
                  <strong>Shortcut:</strong>{" "}
                  <code className="rounded-sm bg-secondary px-1 font-mono">go</code>
                </li>
                <li>
                  <strong>URL:</strong> <BaseUrlCode />
                </li>
              </ul>
            </li>
            <li>
              Click <strong>Add</strong>.
            </li>
          </ol>
        </TabsContent>
        <TabsContent value="firefox" className="space-y-3">
          <ol className="list-decimal space-y-2 pl-6 text-sm">
            <li>Open Bookmarks → Manage Bookmarks.</li>
            <li>
              Create a new bookmark with:
              <ul className="ml-4 mt-1 list-disc space-y-1">
                <li>
                  <strong>Name:</strong> GoLinks
                </li>
                <li>
                  <strong>Location:</strong> <BaseUrlCode />
                </li>
                <li>
                  <strong>Keyword:</strong>{" "}
                  <code className="rounded-sm bg-secondary px-1 font-mono">go</code>
                </li>
              </ul>
            </li>
            <li>Save.</li>
          </ol>
        </TabsContent>
        <TabsContent value="safari" className="space-y-3">
          <p className="text-sm">
            Safari doesn't support custom search engines natively. Options:
          </p>
          <ul className="list-disc space-y-1 pl-6 text-sm">
            <li>
              Bookmark <code className="rounded-sm bg-secondary px-1 font-mono">/</code> for quick
              access.
            </li>
            <li>
              Use an extension like <strong>Keyword Search</strong>.
            </li>
          </ul>
        </TabsContent>
      </Tabs>

      <Alert variant="success">
        <AlertTitle>Pro tip</AlertTitle>
        <AlertDescription>
          Once configured, just type <code className="rounded-sm bg-secondary/70 px-1 font-mono">go keyword</code> in your address bar.
        </AlertDescription>
      </Alert>

      <section className="space-y-3">
        <h2 className="flex items-center gap-2 text-xl font-medium">
          <span className="inline-block h-5 w-1 rounded-sm bg-primary" />
          Usage examples
        </h2>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Command</TableHead>
              <TableHead>Description</TableHead>
              <TableHead>Notes</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow>
              <TableCell>
                <code className="font-mono">go docs</code>
              </TableCell>
              <TableCell>Navigate to a fixed URL.</TableCell>
              <TableCell>
                If <code className="font-mono">docs</code> points to{" "}
                <code className="font-mono">https://docs.example.com</code>.
              </TableCell>
            </TableRow>
            <TableRow>
              <TableCell>
                <code className="font-mono">go jira 123</code>
              </TableCell>
              <TableCell>Navigate with a parameter.</TableCell>
              <TableCell>
                Works if <code className="font-mono">jira</code> contains{" "}
                <code className="font-mono">{"{*}"}</code>.
              </TableCell>
            </TableRow>
            <TableRow>
              <TableCell>
                <code className="font-mono">go github myrepo</code>
              </TableCell>
              <TableCell>Dynamic search.</TableCell>
              <TableCell>
                Same <code className="font-mono">{"{*}"}</code> substitution.
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </section>
    </div>
  );
}
