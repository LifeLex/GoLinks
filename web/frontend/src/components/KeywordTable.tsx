import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { KeywordInfo } from "@/lib/api";

type Props = { keywords: KeywordInfo[] };

function formatDate(iso: string) {
  if (!iso) return "";
  try {
    return new Date(iso).toISOString().slice(0, 10);
  } catch {
    return iso;
  }
}

export function KeywordTable({ keywords }: Props) {
  if (keywords.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No keywords yet — add your first one above.
      </p>
    );
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Keyword</TableHead>
          <TableHead>URL</TableHead>
          <TableHead>Created</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {keywords.map((k) => (
          <TableRow key={`${k.word}-${k.created_at}`}>
            <TableCell>
              <code className="rounded-sm bg-secondary px-1.5 py-0.5 font-mono text-sm">
                {k.word}
              </code>
            </TableCell>
            <TableCell className="break-all font-mono text-xs">
              {k.link.startsWith("http://") || k.link.startsWith("https://") ? (
                <a href={k.link} className="text-accent hover:text-primary hover:underline">
                  {k.link}
                </a>
              ) : (
                k.link
              )}
            </TableCell>
            <TableCell className="text-sm text-muted-foreground">
              {formatDate(k.created_at)}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
