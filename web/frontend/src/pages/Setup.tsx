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
import { SectionHeader } from "@/components/SectionHeader";
import { api } from "@/lib/api";

function Mono({ children }: { children: React.ReactNode }) {
  return (
    <code className="rounded-sm border border-border bg-secondary px-1.5 py-0.5 font-mono text-[0.85em]">
      {children}
    </code>
  );
}

function BaseUrlCode() {
  const { data } = useQuery({ queryKey: ["links"], queryFn: api.listLinks });
  return <Mono>{(data?.base_url ?? "") + "/query/%s"}</Mono>;
}

export function SetupPage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-12 sm:px-6 sm:py-16">
      <div className="space-y-12">
        <header className="animate-rise space-y-4">
          <p className="label-mono">Configuration</p>
          <h1 className="display text-4xl font-bold tracking-tight sm:text-5xl">Browser setup</h1>
          <p className="max-w-xl text-base leading-relaxed text-muted-foreground">
            Register GoLinks as a search engine so typing <Mono>go &lt;keyword&gt;</Mono> in the
            address bar redirects to the matching URL.
          </p>
        </header>

        <div className="animate-rise" style={{ animationDelay: "70ms" }}>
          <Tabs defaultValue="chrome">
            <TabsList>
              <TabsTrigger value="chrome">Chrome / Edge</TabsTrigger>
              <TabsTrigger value="firefox">Firefox</TabsTrigger>
              <TabsTrigger value="safari">Safari</TabsTrigger>
            </TabsList>
            <TabsContent value="chrome" className="space-y-3">
              <ol className="list-decimal space-y-2 pl-6 text-sm marker:font-mono marker:text-primary">
                <li>Open Chrome/Edge settings.</li>
                <li>
                  Go to <strong className="font-semibold">Search engine</strong> →{" "}
                  <strong className="font-semibold">Manage search engines and site search</strong>.
                </li>
                <li>
                  Click <strong className="font-semibold">Add</strong> next to &ldquo;Site
                  search&rdquo;.
                </li>
                <li>
                  Fill in:
                  <ul className="ml-4 mt-1 list-disc space-y-1 marker:text-border">
                    <li>
                      <strong className="font-semibold">Name:</strong> GoLinks
                    </li>
                    <li>
                      <strong className="font-semibold">Shortcut:</strong> <Mono>go</Mono>
                    </li>
                    <li>
                      <strong className="font-semibold">URL:</strong> <BaseUrlCode />
                    </li>
                  </ul>
                </li>
                <li>
                  Click <strong className="font-semibold">Add</strong>.
                </li>
              </ol>
            </TabsContent>
            <TabsContent value="firefox" className="space-y-3">
              <ol className="list-decimal space-y-2 pl-6 text-sm marker:font-mono marker:text-primary">
                <li>Open Bookmarks → Manage Bookmarks.</li>
                <li>
                  Create a new bookmark with:
                  <ul className="ml-4 mt-1 list-disc space-y-1 marker:text-border">
                    <li>
                      <strong className="font-semibold">Name:</strong> GoLinks
                    </li>
                    <li>
                      <strong className="font-semibold">Location:</strong> <BaseUrlCode />
                    </li>
                    <li>
                      <strong className="font-semibold">Keyword:</strong> <Mono>go</Mono>
                    </li>
                  </ul>
                </li>
                <li>Save.</li>
              </ol>
            </TabsContent>
            <TabsContent value="safari" className="space-y-3">
              <p className="text-sm">
                Safari doesn&rsquo;t support custom search engines natively. Options:
              </p>
              <ul className="list-disc space-y-1 pl-6 text-sm marker:text-border">
                <li>
                  Bookmark <Mono>/</Mono> for quick access.
                </li>
                <li>
                  Use an extension like <strong className="font-semibold">Keyword Search</strong>.
                </li>
              </ul>
            </TabsContent>
          </Tabs>
        </div>

        <Alert variant="success" className="animate-rise" style={{ animationDelay: "130ms" }}>
          <AlertTitle>Pro tip</AlertTitle>
          <AlertDescription>
            Once configured, just type <Mono>go keyword</Mono> in your address bar.
          </AlertDescription>
        </Alert>

        <section className="animate-rise space-y-4" style={{ animationDelay: "190ms" }}>
          <SectionHeader index="01" title="Usage examples" />
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
                  <code className="font-mono text-foreground">go docs</code>
                </TableCell>
                <TableCell>Navigate to a fixed URL.</TableCell>
                <TableCell className="text-muted-foreground">
                  If <code className="font-mono">docs</code> points to{" "}
                  <code className="font-mono">https://docs.example.com</code>.
                </TableCell>
              </TableRow>
              <TableRow>
                <TableCell>
                  <code className="font-mono text-foreground">go jira 123</code>
                </TableCell>
                <TableCell>Navigate with a parameter.</TableCell>
                <TableCell className="text-muted-foreground">
                  Works if <code className="font-mono">jira</code> contains{" "}
                  <code className="font-mono">{"{*}"}</code>.
                </TableCell>
              </TableRow>
              <TableRow>
                <TableCell>
                  <code className="font-mono text-foreground">go github myrepo</code>
                </TableCell>
                <TableCell>Dynamic search.</TableCell>
                <TableCell className="text-muted-foreground">
                  Same <code className="font-mono">{"{*}"}</code> substitution.
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </section>
      </div>
    </div>
  );
}
