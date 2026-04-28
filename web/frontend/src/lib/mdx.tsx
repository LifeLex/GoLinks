import * as React from "react";
import * as runtime from "react/jsx-runtime";
import { evaluate } from "@mdx-js/mdx";

type MDXModule = { default: React.ComponentType };
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";

import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "@/components/ui/table";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";

// Components exposed to MDX. Authors can write <Alert variant="destructive">...</Alert>
// directly inside their .mdx files and it will render as a shadcn Alert.
const mdxComponents = {
  Alert,
  AlertTitle,
  AlertDescription,
  Button,
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
  // Wire shadcn Table primitives to plain markdown tables so GFM tables pick up the styling.
  table: (props: React.HTMLAttributes<HTMLTableElement>) => <Table {...props} />,
  thead: (props: React.HTMLAttributes<HTMLTableSectionElement>) => <TableHeader {...props} />,
  tbody: (props: React.HTMLAttributes<HTMLTableSectionElement>) => <TableBody {...props} />,
  tr: (props: React.HTMLAttributes<HTMLTableRowElement>) => <TableRow {...props} />,
  th: (props: React.ThHTMLAttributes<HTMLTableCellElement>) => <TableHead {...props} />,
  td: (props: React.TdHTMLAttributes<HTMLTableCellElement>) => <TableCell {...props} />,
};

export async function compileMDX(source: string): Promise<MDXModule> {
  return evaluate(source, {
    ...(runtime as typeof runtime & { Fragment: React.ComponentType }),
    baseUrl: import.meta.url,
    remarkPlugins: [remarkGfm],
    rehypePlugins: [rehypeHighlight],
    useMDXComponents: () => mdxComponents,
  });
}
