export type KeywordInfo = {
  word: string;
  link: string;
  created_at: string;
  tags: string[] | null;
};

export type SearchResponse = {
  query: string;
  results: KeywordInfo[];
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
  tags?: string[];
};

export type Role = "admin" | "user";

export type User = {
  id: number;
  email: string;
  role: Role;
  created_at: string;
};

export type AuthStatus = {
  needs_setup: boolean;
  authenticated: boolean;
  user?: User | null;
};

export type Credentials = {
  email: string;
  password: string;
};

export type CreateUserInput = {
  email: string;
  password: string;
  role: Role;
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
    // Send the session cookie on every request (same-origin SPA).
    credentials: "include",
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
  updateLink: (word: string, input: { link: string; tags?: string[] }) =>
    request<{ success: true }>(`/api/links/${encodeURIComponent(word)}`, {
      method: "PATCH",
      body: JSON.stringify(input),
    }),
  deleteLink: (word: string) =>
    request<{ success: true }>(`/api/links/${encodeURIComponent(word)}`, {
      method: "DELETE",
    }),
  search: (query: string, limit?: number) => {
    const params = new URLSearchParams({ q: query });
    if (limit) params.set("limit", String(limit));
    return request<SearchResponse>(`/api/search?${params.toString()}`);
  },
  listDocs: () => request<{ documents: DocumentInfo[] }>("/api/docs"),
  getDoc: (filename: string) =>
    request<DocumentSource>(`/api/docs/${encodeURIComponent(filename)}`),
  uploadDoc: async (file: File) => {
    const form = new FormData();
    form.append("file", file);
    const res = await fetch("/api/docs", {
      method: "POST",
      body: form,
      credentials: "include",
    });
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

  // --- auth ---
  authStatus: () => request<AuthStatus>("/auth/status"),
  setup: (creds: Credentials) =>
    request<{ user: User }>("/auth/setup", {
      method: "POST",
      body: JSON.stringify(creds),
    }),
  login: (creds: Credentials) =>
    request<{ user: User }>("/auth/login", {
      method: "POST",
      body: JSON.stringify(creds),
    }),
  logout: () => request<{ success: true }>("/auth/logout", { method: "POST" }),
  listUsers: () => request<{ users: User[] }>("/api/users"),
  createUser: (input: CreateUserInput) =>
    request<{ user: User }>("/api/users", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  deleteUser: (id: number) =>
    request<{ success: true }>(`/api/users/${id}`, { method: "DELETE" }),
};

export { ApiError };
