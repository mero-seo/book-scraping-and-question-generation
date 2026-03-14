import type {
  AllowedSource,
  Book,
  Chapter,
  PaginatedResponse,
  Question,
  SearchResult,
  User,
  UserAnswer,
  UserStats,
} from './types';

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// ---------------------------------------------------------------------------
// Token management
// ---------------------------------------------------------------------------

let authToken: string | null = null;

export function setToken(token: string) {
  authToken = token;
  if (typeof window !== 'undefined') {
    localStorage.setItem('token', token);
  }
}

export function getToken(): string | null {
  if (authToken) return authToken;
  if (typeof window !== 'undefined') {
    authToken = localStorage.getItem('token');
  }
  return authToken;
}

export function clearToken() {
  authToken = null;
  if (typeof window !== 'undefined') {
    localStorage.removeItem('token');
  }
}

// ---------------------------------------------------------------------------
// Error class
// ---------------------------------------------------------------------------

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public code?: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

// ---------------------------------------------------------------------------
// Generic fetch wrapper
// ---------------------------------------------------------------------------

async function apiFetch<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((options.headers as Record<string, string>) || {}),
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_BASE}${path}`, { ...options, headers });

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new ApiError(res.status, body.error || res.statusText, body.code);
  }

  // 204 No Content — nothing to parse
  if (res.status === 204) return undefined as T;

  return res.json();
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Build a URLSearchParams from an object, mapping camelCase keys to
 *  snake_case query parameters where the backend expects it. */
function buildQuery(
  params: Record<string, string | number | boolean | undefined> | undefined,
  keyMap: Record<string, string> = {},
): string {
  if (!params) return '';
  const qs = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined) continue;
    const queryKey = keyMap[key] ?? key;
    qs.set(queryKey, String(value));
  }
  const str = qs.toString();
  return str ? `?${str}` : '';
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

export async function register(data: {
  email: string;
  name: string;
  password: string;
  gradeLevel?: string;
  educationSystem?: string;
  examPreparingFor?: string;
}) {
  return apiFetch<{ token: string; user: User }>('/api/v1/auth/register', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function login(email: string, password: string) {
  return apiFetch<{ token: string; user: User }>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}

// ---------------------------------------------------------------------------
// Books
// ---------------------------------------------------------------------------

export async function listBooks(params?: {
  page?: number;
  limit?: number;
  status?: string;
  subject?: string;
  gradeLevel?: string;
  sourceType?: string;
  sort?: string;
  order?: string;
}) {
  const qs = buildQuery(params, {
    gradeLevel: 'grade_level',
    sourceType: 'source_type',
  });
  return apiFetch<PaginatedResponse<Book>>(`/api/v1/books${qs}`);
}

export async function getBook(id: string) {
  return apiFetch<Book>(`/api/v1/books/${id}`);
}

export async function createBook(data: {
  sourceUrl: string;
  sourceType: string;
  subject: string;
  gradeLevels: string[];
  title?: string;
  author?: string;
}) {
  return apiFetch<Book>('/api/v1/books', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateBook(id: string, data: Partial<Book>) {
  return apiFetch<Book>(`/api/v1/books/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export async function deleteBook(id: string) {
  return apiFetch<{ message: string; deleted: Record<string, number> }>(
    `/api/v1/books/${id}`,
    { method: 'DELETE' },
  );
}

export async function processBook(id: string) {
  return apiFetch<{ message: string; bookId: string; status: string }>(
    `/api/v1/books/${id}/process`,
    { method: 'POST' },
  );
}

export async function getBookStatus(id: string) {
  return apiFetch<{
    bookId: string;
    status: string;
    processingError?: string;
    chaptersTotal: number;
    chaptersEmbedded: number;
    questionsGenerated: number;
  }>(`/api/v1/books/${id}/status`);
}

export async function searchBooks(query: string, limit?: number) {
  return apiFetch<{ results: SearchResult[] }>('/api/v1/books/search', {
    method: 'POST',
    body: JSON.stringify({ query, limit }),
  });
}

// ---------------------------------------------------------------------------
// Chapters
// ---------------------------------------------------------------------------

export async function listChapters(
  bookId: string,
  params?: { page?: number; limit?: number; includeContent?: boolean },
) {
  const qs = buildQuery(params, { includeContent: 'include_content' });
  return apiFetch<PaginatedResponse<Chapter>>(
    `/api/v1/books/${bookId}/chapters${qs}`,
  );
}

export async function getChapter(bookId: string, chapterId: string) {
  return apiFetch<Chapter>(
    `/api/v1/books/${bookId}/chapters/${chapterId}`,
  );
}

export async function searchChapters(
  bookId: string,
  query: string,
  limit?: number,
) {
  return apiFetch<{ results: Array<{ chapter: Chapter; score: number }> }>(
    `/api/v1/books/${bookId}/chapters/search`,
    { method: 'POST', body: JSON.stringify({ query, limit }) },
  );
}

// ---------------------------------------------------------------------------
// Questions
// ---------------------------------------------------------------------------

export async function listQuestions(params?: {
  page?: number;
  limit?: number;
  bookId?: string;
  chapterId?: string;
  questionType?: string;
  bloomLevel?: string;
  difficulty?: string;
}) {
  const qs = buildQuery(params, {
    bookId: 'book_id',
    chapterId: 'chapter_id',
    questionType: 'question_type',
    bloomLevel: 'bloom_level',
  });
  return apiFetch<PaginatedResponse<Question>>(`/api/v1/questions${qs}`);
}

export async function getQuestion(id: string) {
  return apiFetch<Question>(`/api/v1/questions/${id}`);
}

export async function getRandomQuestions(params?: {
  count?: number;
  bookId?: string;
  chapterId?: string;
  questionType?: string;
  bloomLevel?: string;
  difficulty?: string;
}) {
  const qs = buildQuery(params, {
    bookId: 'book_id',
    chapterId: 'chapter_id',
    questionType: 'question_type',
    bloomLevel: 'bloom_level',
  });
  return apiFetch<{ data: Question[] }>(`/api/v1/questions/random${qs}`);
}

// ---------------------------------------------------------------------------
// Answers
// ---------------------------------------------------------------------------

export async function submitAnswer(data: {
  questionId: string;
  answerText: string;
  timeTaken?: number;
}) {
  return apiFetch<UserAnswer>('/api/v1/answers', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function getAnswerHistory(params?: {
  page?: number;
  limit?: number;
}) {
  const qs = buildQuery(params);
  return apiFetch<PaginatedResponse<UserAnswer>>(`/api/v1/answers${qs}`);
}

export async function getUserStats() {
  return apiFetch<UserStats>('/api/v1/answers/stats');
}

// ---------------------------------------------------------------------------
// Admin — dashboard
// ---------------------------------------------------------------------------

export async function getDashboard() {
  return apiFetch<Record<string, unknown>>('/api/v1/admin/dashboard');
}

// ---------------------------------------------------------------------------
// Admin — allowed sources
// ---------------------------------------------------------------------------

export async function listSources(params?: {
  page?: number;
  limit?: number;
}) {
  const qs = buildQuery(params);
  return apiFetch<PaginatedResponse<AllowedSource>>(
    `/api/v1/admin/sources${qs}`,
  );
}

export async function createSource(data: {
  urlPattern: string;
  name: string;
  sourceType: string;
  notes?: string;
}) {
  return apiFetch<AllowedSource>('/api/v1/admin/sources', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateSource(id: string, data: Partial<AllowedSource>) {
  return apiFetch<AllowedSource>(`/api/v1/admin/sources/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export async function deleteSource(id: string) {
  return apiFetch<{ message: string }>(`/api/v1/admin/sources/${id}`, {
    method: 'DELETE',
  });
}

// ---------------------------------------------------------------------------
// Admin — users
// ---------------------------------------------------------------------------

export async function listUsers(params?: {
  page?: number;
  limit?: number;
}) {
  const qs = buildQuery(params);
  return apiFetch<PaginatedResponse<User>>(`/api/v1/admin/users${qs}`);
}

export async function updateUserRole(id: string, role: string) {
  return apiFetch<{ message: string; role: string }>(
    `/api/v1/admin/users/${id}/role`,
    { method: 'PUT', body: JSON.stringify({ role }) },
  );
}
