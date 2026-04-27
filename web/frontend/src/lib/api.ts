export type KeywordInfo = {
  word: string;
  link: string;
  created_at: string;
};

export type PopularQuery = {
  count: number;
  word: string;
  link: string;
};

export type LinksResponse = {
  keywords: KeywordInfo[];
  recent_queries: PopularQuery[];
  base_url: string;
};

export type DocumentInfo = {
  title: string;
  description?: string;
  type: "markdown" | "mdx";
  path: string;
};

export type DocumentSource = {
  source: string;
  type: "markdown" | "mdx";
  metadata: DocumentInfo;
};

export type LinkInput = {
  word: string;
  link: string;
};

class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
    ...init,
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new ApiError(text || res.statusText, res.status);
  }

  // DELETE can return empty
  const ct = res.headers.get("content-type") ?? "";
  if (!ct.includes("application/json")) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

export const api = {
  listLinks: () => request<LinksResponse>("/api/links"),
  createLink: (input: LinkInput) =>
    request<{ success: true }>("/api/links", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  listDocs: () => request<{ documents: DocumentInfo[] }>("/api/docs"),
  getDoc: (filename: string) =>
    request<DocumentSource>(`/api/docs/${encodeURIComponent(filename)}`),
  uploadDoc: async (file: File) => {
    const form = new FormData();
    form.append("file", file);
    const res = await fetch("/api/docs", { method: "POST", body: form });
    if (!res.ok) throw new ApiError(await res.text(), res.status);
    return (await res.json()) as {
      success: true;
      filename: string;
      url: string;
    };
  },
  deleteDoc: (filename: string) =>
    request<{ success: true }>(`/api/docs/${encodeURIComponent(filename)}`, {
      method: "DELETE",
    }),
};

export { ApiError };
